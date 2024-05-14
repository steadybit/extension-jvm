package test

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func RequireProcessEmitted(t *testing.T, ch <-chan *process.Process, pid int32) *process.Process {
	t.Helper()

	for {
		select {
		case p, ok := <-ch:
			fmt.Printf("Process %v\n", p)
			if ok && p.Pid == pid {
				return p
			}

		case <-time.After(5 * time.Second):
			require.Failf(t, "JVM not discovered", "JVM with PID %d not discovered", pid)
			return nil
		}
	}
}

func AssertProcessEmitted(t *testing.T, ch <-chan *process.Process, pid int32) bool {
	t.Helper()

	for {
		select {
		case p, ok := <-ch:
			log.Info().Msgf("New JVM Process emitted %v\n", p)
			if ok && p.Pid == pid {
				return true
			}

		case <-time.After(10 * time.Second):
			return assert.Failf(t, "JVM not discovered", "JVM with PID %d not discovered", pid)
		}
	}
}
