package extjvm

import "github.com/steadybit/extension-jvm/extjvm/attachment"

type JvmAttachment interface {
	Attach(agentJar string, initJar string, port int) bool
	CopyFiles(dstPath string, files map[string]string)
	GetAgentHost() string
}

func GetAttachment(jvm *JavaVm) JvmAttachment {
	if jvm.IsRunningInContainer() {
		return attachment.ContainerJvmAttachment{}
	}
	return attachment.HostJvmAttachment{}
}
