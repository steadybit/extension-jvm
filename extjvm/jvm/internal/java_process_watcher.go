// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package internal

import (
	"context"
	"github.com/procyon-projects/chrono"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ProcessWatcher struct {
	seenCreateTime int64
	scheduler      chrono.TaskScheduler
	Processes      <-chan *process.Process
	ch             chan<- *process.Process
}

func (w *ProcessWatcher) Start() {
	w.StartWithInterval(5 * time.Second)
}

func (w *ProcessWatcher) StartWithInterval(interval time.Duration) {
	w.scheduler = chrono.NewDefaultTaskScheduler()
	ch := make(chan *process.Process)
	w.ch = ch
	w.Processes = ch

	if _, err := w.scheduler.ScheduleWithFixedDelay(func(ctx context.Context) { w.lookForNewProcesses(ctx) }, interval); err == nil {
		log.Info().Msg("Watching for new Java processes.")
	} else {
		log.Error().Err(err).Msg("Failed to schedule Java Process Watcher Task.")
	}
}

func (w *ProcessWatcher) Stop() {
	log.Info().Msg("Stopped watching for new Java processes")
	w.scheduler.Shutdown()
	close(w.ch)
}

func (w *ProcessWatcher) lookForNewProcesses(ctx context.Context) {
	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list processes")
		return
	}

	count := 0
	lastSeenCreateTime := w.seenCreateTime
	maxCreateTime := time.Now().Add(-1 * time.Second).UnixMilli() // we want to ignore processes created in the last second
	for _, p := range processes {
		createTime, _ := p.CreateTimeWithContext(ctx)
		if createTime <= lastSeenCreateTime || createTime > maxCreateTime {
			continue
		}

		if createTime > w.seenCreateTime {
			w.seenCreateTime = createTime
		}

		if !isJavaProcess(ctx, p) {
			continue
		}

		count++
		w.ch <- p
	}

	log.Trace().Msgf("Found %d new Java processes", count)
}

func isJavaProcess(ctx context.Context, p *process.Process) bool {
	return strings.HasSuffix(getProcessExe(ctx, p), "java")
}

func getProcessExe(ctx context.Context, p *process.Process) string {
	exePath := filepath.Join("/proc", strconv.Itoa(int(p.Pid)), "exe")
	output, err := utils.RootCommandContext(ctx, "readlink", exePath).Output()
	if err == nil {
		return strings.TrimSpace(string(output))
	}

	exe, err := p.Exe()
	if err == nil {
		return exe
	}

	return ""
}
