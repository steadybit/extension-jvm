package heartbeat

import (
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

type Heartbeat struct {
	file        string
	fileCreated bool
	interval    time.Duration
	done        chan struct{}
}

func NewHeartbeat(file string, interval time.Duration) *Heartbeat {
	return &Heartbeat{
		interval: interval,
		file:     file,
	}
}

func (h *Heartbeat) File() string {
	if h.fileCreated {
		return h.file
	}
	return ""
}

func (h *Heartbeat) Start() error {
	if err := h.createFile(); err != nil {
		return err
	}

	h.fileCreated = true
	h.done = make(chan struct{}, 1)

	go func(h *Heartbeat) {
		tick := time.Tick(h.interval)
		for {
			select {
			case <-tick:
				h.touchFile()
			case <-h.done:
				return
			}
		}
	}(h)

	return nil
}

func (h *Heartbeat) Stop() {
	if h.done != nil {
		h.done <- struct{}{}
		close(h.done)
	}
	h.deleteFile()
}

func (h *Heartbeat) touchFile() {
	now := time.Now()
	if err := os.Chtimes(h.file, now, now); err != nil {
		log.Warn().Err(err).Str("file", h.file).Msg("Failed to touch heartbeat file")
	}
}

func (h *Heartbeat) createFile() error {
	log.Trace().Str("file", h.file).Msg("Creating file for javaagent heartbeat")
	f, err := os.Create(h.file)
	if err != nil {
		return err
	}

	defer func(f *os.File) {
		_ = f.Close()
	}(f)

	_, err = fmt.Fprintf(f, "%d\n", os.Getpid())
	return err
}

func (h *Heartbeat) deleteFile() {
	if !h.fileCreated {
		return
	}
	log.Trace().Str("file", h.file).Msg("Removing file for javaagent heartbeat")
	if err := os.Remove(h.file); err != nil {
		log.Warn().Err(err).Str("file", h.file).Msg("Failed to remove heartbeat file")
	}
	h.fileCreated = false
}
