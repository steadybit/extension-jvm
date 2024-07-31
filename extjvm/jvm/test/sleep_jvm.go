// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package test

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/process"
	"os/exec"
	"strings"
)

const src = `
				var millis = 60_000L;
				System.out.printf("Sleeping for %dms",millis);
        Thread.sleep(millis);
`

type SleepJvm struct {
	cmd *exec.Cmd
}

func NewSleep() *SleepJvm {
	cmd := exec.Command("java", "/Users/jedmeier/projects/steadybit/extension-jvm/extjvm/jvm/test/SleepJvm.java", "60000") //FIXME
	cmd.Stdin = strings.NewReader(src)
	if err := cmd.Start(); err != nil {
		panic(err)
	} else {
		fmt.Printf("Started new JVM with PID %d\n", cmd.Process.Pid)
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
