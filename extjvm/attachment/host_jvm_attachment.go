package attachment


type HostJvmAttachment struct {}

func (attachment HostJvmAttachment) Attach(agentJar string, initJar string, port int) bool {
 //TODO: implement
  return false
}

func (attachment HostJvmAttachment) CopyFiles(dstPath string, files map[string]string) {
  //TODO: implement
}

func (attachment HostJvmAttachment) GetAgentHost() string {
  //TODO: implement
  return ""
}
