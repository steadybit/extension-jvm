// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package test

import (
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v3/process"
	"os"
	"os/exec"
	"sync"
)

const src = `
import java.time.Instant;

class SleepJvm {
	public static void main(String[] args) throws Exception {
		var pid = ProcessHandle.current().pid();
		var millis = Long.parseLong(args[0]);
		System.out.printf("[%d] %s\tSleeping for %dms\n", pid, Instant.now(), millis);
		Thread.sleep(millis);
	}
}
`

var (
	srcTmpFileOnce = sync.Once{}
	srcTmpFile     string
)

type SleepJvm struct {
	cmd *exec.Cmd
}

func NewSleep() *SleepJvm {
	srcTmpFileOnce.Do(func() {
		f, err := os.CreateTemp("", "SleepJvm_*.java")
		if err != nil {
			panic(err)
		}
		srcTmpFile = f.Name()
		_, _ = f.WriteString(src)
		_ = f.Close()
	})

	cmd := exec.Command("java", srcTmpFile, "60000")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		panic(err)
	} else {
		log.Info().Msgf("Started new JVM with PID %d\n", cmd.Process.Pid)
		return &SleepJvm{cmd: cmd}
	}
}

func (s *SleepJvm) Stop() {
	err := s.cmd.Process.Kill()
	if err == nil {
		_, _ = s.cmd.Process.Wait()
	}
}

func (s *SleepJvm) Pid() int32 {
	return int32(s.cmd.Process.Pid)
}

func (s *SleepJvm) Process() *process.Process {
	p, err := process.NewProcess(int32(s.cmd.Process.Pid))
	if err != nil {
		panic(err)
	}
	return p
}
