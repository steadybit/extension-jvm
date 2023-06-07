package attachment


type ContainerJvmAttachment struct {}

func (attachment ContainerJvmAttachment) Attach(agentJar string, initJar string, port int) bool {
 //TODO: implement
  return false
}

func (attachment ContainerJvmAttachment) CopyFiles(dstPath string, files map[string]string) {
  //TODO: implement
}

func (attachment ContainerJvmAttachment) GetAgentHost() string {
  //TODO: implement
  return ""
}
