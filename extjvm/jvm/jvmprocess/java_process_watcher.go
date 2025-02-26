// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package jvmprocess

import (
	"codnect.io/chrono"
	"context"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/steadybit/extension-jvm/chrono_utils"
	"github.com/steadybit/extension-jvm/extjvm/jvm/starttime"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ProcessWatcher struct {
	seenStartTime starttime.Time
	scheduler     chrono.TaskScheduler
	Processes     <-chan *process.Process
	ch            chan<- *process.Process
	Interval      time.Duration
}

func (w *ProcessWatcher) Start() {
	w.scheduler = chrono_utils.NewContextTaskScheduler()

	ch := make(chan *process.Process)
	w.ch = ch
	w.Processes = ch

	if _, err := w.scheduler.ScheduleWithFixedDelay(w.lookForNewProcesses, w.Interval); err == nil {
		log.Info().Msg("Watching for new Java processes.")
	} else {
		log.Error().Err(err).Msg("Failed to schedule Java Process Watcher Task.")
	}
}

func (w *ProcessWatcher) Stop() {
	log.Info().Msg("Stopped watching for new Java processes")
	<-w.scheduler.Shutdown()
	close(w.ch)
}

func (w *ProcessWatcher) lookForNewProcesses(ctx context.Context) {
	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list processes")
		return
	}

	lastSeenStartTime := w.seenStartTime
	for _, p := range processes {
		if !isJavaProcess(ctx, p) {
			continue
		}

		startTime, _ := starttime.ForProcess(p)
		if !startTime.After(lastSeenStartTime) {
			log.Trace().Msgf("Ignoring java process with PID %d", p.Pid)
			continue
		}

		if startTime.After(w.seenStartTime) {
			w.seenStartTime = startTime
		}

		log.Trace().Msgf("Found new java processe with PID %d", p.Pid)
		select {
		case w.ch <- p:
		case <-ctx.Done():
		}
	}

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
