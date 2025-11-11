package jvm

import (
	"path"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/utils"
)

type hostJvmAttachment struct {
	jvm JavaVm
}

func (a hostJvmAttachment) attach(agentHTTPPort int, heartbeatFile string) bool {
	if !a.jvm.IsRunning() {
		log.Debug().Msgf("Process not running. Skipping a to JVM %s", a.jvm.ToDebugString())
		return false
	}

	resolvedMainJar, _ := a.resolveFile(mainJarName)
	resolvedInitJar, _ := a.resolveFile(initJarName)
	resolvedHeartBeatFile, _ := a.resolveFile(heartbeatFile)
	return externalAttach(a.jvm,
		resolvedMainJar,
		resolvedInitJar,
		resolvedHeartBeatFile,
		agentHTTPPort,
		a.GetHostAddress(),
		a.jvm.Pid(),
		a.jvm.Pid(),
		utils.RootCommandContext,
	)
}

func (a hostJvmAttachment) resolveFile(f string) (string, error) {
	return path.Join(javaagentPath(), f), nil
}

func (a hostJvmAttachment) GetHostAddress() string {
	return "127.0.0.1"
}
