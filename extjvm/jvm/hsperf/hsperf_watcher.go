// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package hsperf

import (
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/process"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

var (
	regexIsHsperfdata = regexp.MustCompile(`hsperfdata_[^/\\]`)
)

type Watcher struct {
	watcher   *fsnotify.Watcher
	Processes <-chan *process.Process
}

func (w *Watcher) Start() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Error().Err(err).Msgf("Failed to watch for hsperfdata")
	}
	w.watcher = watcher
	ch := make(chan *process.Process)
	w.Processes = ch

	go func(ch chan<- *process.Process) {
		defer close(ch)

		w.walkHsperfdataDir(os.TempDir(), ch)

		for {
			select {
			case event, ok := <-w.watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Create) {
					w.walkHsperfdataDir(event.Name, ch)
				}
			case err, ok := <-w.watcher.Errors:
				if !ok {
					return
				}
				log.Warn().Err(err).Msg("error from hsperfdata watcher")
			}
		}
	}(ch)
}

func (w *Watcher) Stop() {
	if w.watcher != nil {
		log.Info().Msg("Stopped watching for hsperfdata")
		err := w.watcher.Close()
		if err != nil {
			log.Warn().Err(err).Msg("Failed to close hsperfdata watcher")
		}
	}
}

func (w *Watcher) walkHsperfdataDir(path string, ch chan<- *process.Process) {
	if err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if !regexIsHsperfdata.MatchString(path) {
			return nil
		}

		if err != nil {
			log.Warn().Err(err).Msgf("Watching directory %s for hsperfdata failed", path)
			return nil
		}

		if info.IsDir() {
			if err := w.watcher.Add(path); err == nil {
				log.Info().Msgf("Watching directory %s for hsperfdata", path)
			} else {
				log.Warn().Err(err).Msgf("Watching directory %s for hsperfdata failed", path)
			}
			return nil
		}

		pid, err := strconv.Atoi(filepath.Base(path))
		if err != nil {
			log.Trace().Err(err).Msgf("Failed to parse pid from %s not a hsperfdata file", path)
			return nil
		}

		p, err := process.NewProcess(int32(pid))
		if err == nil {
			ch <- p
		} else {
			log.Trace().Err(err).Msgf("Failed to get process %d", pid)
		}
		return nil
	}); err != nil {
		log.Error().Err(err).Msgf("Failed to watch for hsperfdata in %s", path)
	}
}
