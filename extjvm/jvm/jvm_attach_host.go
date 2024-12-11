package jvm

import (
	"github.com/rs/zerolog/log"
)

type hostJvmAttachment struct {
	jvm JavaVm
}

func (a hostJvmAttachment) attach(agentJar string, initJar string, agentHTTPPort int) bool {
	if !a.jvm.IsRunning() {
		log.Debug().Msgf("Process not running. Skipping a to JVM %s", a.jvm.ToDebugString())
		return false
	}
	return externalAttach(a.jvm, agentJar, initJar, agentHTTPPort, a.GetHostAddress(), a.jvm.Pid(), a.jvm.Pid(), "")
}

func (a hostJvmAttachment) canAccessHostFiles() bool {
	return true
}

func (a hostJvmAttachment) copyFiles(_ string, _ map[string]string) (map[string]string, error) {
	panic("not supported")
}

func (a hostJvmAttachment) GetHostAddress() string {
	return "127.0.0.1"
}
