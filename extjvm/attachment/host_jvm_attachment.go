package attachment

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/java_process"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"strconv"
)

type HostJvmAttachment struct {
	Jvm *jvm.JavaVm
}

func (attachment HostJvmAttachment) Attach(agentJar string, initJar string, agentHTTPPort int) bool {
	if !java_process.IsRunningProcess(attachment.Jvm.Pid) {
		log.Debug().Msgf("Process not running. Skipping attachment to JVM %+v", attachment.Jvm)
		return false
	}
	return externalAttach(attachment.Jvm, agentJar, initJar, agentHTTPPort, attachment.GetAgentHost(), false, strconv.Itoa(int(attachment.Jvm.Pid)), strconv.Itoa(int(attachment.Jvm.Pid)))
}

func (attachment HostJvmAttachment) CopyFiles(_ string, _ map[string]string) (map[string]string, error) {
	panic("not supported")
}

func (attachment HostJvmAttachment) GetAgentHost() string {
	return "127.0.0.1"
}
