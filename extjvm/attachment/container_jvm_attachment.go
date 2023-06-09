package attachment

import (
  "github.com/rs/zerolog/log"
  "github.com/steadybit/extension-jvm/extjvm/java_process"
  "github.com/steadybit/extension-jvm/extjvm/jvm"
)

type ContainerJvmAttachment struct {
  Jvm *jvm.JavaVm
}

func (attachment ContainerJvmAttachment) Attach(agentJar string, initJar string, agentHttpPort int) bool {
  if !java_process.IsRunningProcess(attachment.Jvm.Pid) {
    log.Debug().Msgf("Process not running. Skipping attachment to JVM %+v", attachment.Jvm)
    return false
  }
  return externalAttach(attachment.Jvm, agentJar, initJar, agentHttpPort, attachment.GetAgentHost(), true)
}

func (attachment ContainerJvmAttachment) CopyFiles(dstPath string, files map[string]string) {
  //via root cmd
  //TODO: implement
}

func (attachment ContainerJvmAttachment) GetAgentHost() string {
  return "127.0.0.1"
}
