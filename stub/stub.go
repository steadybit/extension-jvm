/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package main

import (
	"bufio"
	"fmt"
	"github.com/steadybit/extension-jvm/config"
	"github.com/steadybit/extension-jvm/extjvm"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-kit/extlogging"
	"io"
	"os"
	"strconv"
	"strings"
)

func main() {
	extlogging.InitZeroLog()
	config.ParseConfiguration()
	config.Config.JavaAgentLogLevel = "TRACE"

	stop, facade, _, _ := extjvm.StartJvmInfrastructure()
	defer stop()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("> ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSuffix(text, "\n")
		if text == "exit" {
			break
		}
		if text == "list" {
			for _, vm := range facade.GetJvms() {
				fmt.Printf("%s\n", vm.ToInfoString())
			}
			continue
		}

		pidCmdArg := strings.SplitN(text, " ", 2)
		if len(pidCmdArg) < 2 {
			fmt.Printf("[pid] [command]:[args...]\n")
			continue
		}
		pid, _ := strconv.Atoi(pidCmdArg[0])
		cmdArg := strings.SplitN(pidCmdArg[1], ":", 2)
		if len(cmdArg) < 2 {
			fmt.Printf("%s [command]:[args...]\n", pidCmdArg[0])
			continue
		}

		if javaVm := facade.GetJvm(int32(pid)); javaVm != nil {
			if s, err := facade.SendCommandToAgentWithHandler(javaVm, cmdArg[0], cmdArg[1], toString); err != nil {
				fmt.Printf("Error: %s\n", err)
			} else {
				fmt.Printf("%s\n", s)
			}
		} else {
			fmt.Printf("JVM with pid %s not found\n", pidCmdArg[0])
		}
	}
}

func toString(response io.Reader) (any, error) {
	return jvm.GetCleanSocketCommandResult(response)
}
