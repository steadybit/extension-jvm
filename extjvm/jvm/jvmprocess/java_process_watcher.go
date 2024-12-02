// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package jvmprocess

import (
	"codnect.io/chrono"
	"context"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ProcessWatcher struct {
	seenStartTime int64 //start time of the last seen process in clock ticks
	scheduler     chrono.TaskScheduler
	Processes     <-chan *process.Process
	ch            chan<- *process.Process
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
	<-w.scheduler.Shutdown()
	close(w.ch)
}

func (w *ProcessWatcher) lookForNewProcesses(ctx context.Context) {
	processes, err := process.ProcessesWithContext(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list processes")
		return
	}

	count := 0
	lastSeenStartTime := w.seenStartTime
	for _, p := range processes {
		if !isJavaProcess(ctx, p) {
			continue
		}

		startTime, _ := p.CreateTime()
		if startTime <= lastSeenStartTime {
			log.Trace().
				Int64("startTime", startTime).
				Int64("lastSeenStartTime", lastSeenStartTime).
				Msgf("Ignoring java process with PID %d", p.Pid)
			continue
		}

		if startTime > w.seenStartTime {
			w.seenStartTime = startTime
		}

		log.Trace().Msgf("Found new java processes with PID %d", p.Pid)
		count++
		w.ch <- p
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
