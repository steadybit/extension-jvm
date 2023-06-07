package attachment

import (
	"github.com/steadybit/extension-jvm/extjvm"
)

type JvmAttachment interface {
	Attach(agentJar string, initJar string, port int) bool
	CopyFiles(dstPath string, files map[string]string)
	GetAgentHost() string
}

func GetAttachment(jvm *extjvm.JavaVm) JvmAttachment {
	if jvm.IsRunningInContainer() {
		return ContainerJvmAttachment{}
	}
	return HostJvmAttachment{}
}
