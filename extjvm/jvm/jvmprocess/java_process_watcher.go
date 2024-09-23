// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package jvmprocess

import (
	"bytes"
	"codnect.io/chrono"
	"context"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"os"
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

		startTime, _ := startTime(p)
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

// When https://github.com/shirou/gopsutil/pull/1713 is resolved we can remove this code and use the create time of the
// process instead. But currently the lacking precision causes undiscovered processes.
func startTime(p *process.Process) (int64, error) {
	statPath := filepath.Join("/proc", strconv.Itoa(int(p.Pid)), "stat")
	contents, err := os.ReadFile(statPath)
	if err != nil {
		return 0, err
	}
	fields := splitProcStat(contents)
	return strconv.ParseInt(fields[22], 10, 64)
}

func splitProcStat(content []byte) []string {
	nameStart := bytes.IndexByte(content, '(')
	nameEnd := bytes.LastIndexByte(content, ')')
	restFields := strings.Fields(string(content[nameEnd+2:])) // +2 skip ') '
	name := content[nameStart+1 : nameEnd]
	pid := strings.TrimSpace(string(content[:nameStart]))
	fields := make([]string, 3, len(restFields)+3)
	fields[1] = pid
	fields[2] = string(name)
	fields = append(fields, restFields...)
	return fields
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
