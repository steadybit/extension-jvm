package jvm

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/java_process"
	"strconv"
)

type hostJvmAttachment struct {
	jvm *JavaVm
}

func (a hostJvmAttachment) attach(agentJar string, initJar string, agentHTTPPort int) bool {
	if !java_process.IsRunningProcess(a.jvm.Pid) {
		log.Debug().Msgf("Process not running. Skipping a to JVM %+v", a.jvm)
		return false
	}
	return externalAttach(a.jvm, agentJar, initJar, agentHTTPPort, a.GetHostAddress(), false, strconv.Itoa(int(a.jvm.Pid)), strconv.Itoa(int(a.jvm.Pid)))
}

func (a hostJvmAttachment) copyFiles(_ string, _ map[string]string) (map[string]string, error) {
	panic("not supported")
}

func (a hostJvmAttachment) GetHostAddress() string {
	return "127.0.0.1"
}
