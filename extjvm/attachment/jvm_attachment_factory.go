package attachment

import (
  "github.com/steadybit/extension-jvm/extjvm/jvm"
)

type JvmAttachment interface {
	Attach(agentJar string, initJar string, port int) bool
	CopyFiles(dstPath string, files map[string]string)
	GetAgentHost() string
}

func GetAttachment(jvm *jvm.JavaVm) JvmAttachment {
	if jvm.IsRunningInContainer() {
		return ContainerJvmAttachment{
      Jvm: jvm,
    }
	}
	return HostJvmAttachment{
    Jvm: jvm,
  }
}
