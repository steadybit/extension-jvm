package attachment

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/java_process"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

type HostJvmAttachment struct {
	Jvm *jvm.JavaVm
}

func (attachment HostJvmAttachment) Attach(agentJar string, initJar string, agentHttpPort int) bool {
	if !java_process.IsRunningProcess(attachment.Jvm.Pid) {
		log.Debug().Msgf("Process not running. Skipping attachment to JVM %+v", attachment.Jvm)
		return false
	}
	return attachment.externalAttach(agentJar, initJar, agentHttpPort)
}

func (attachment HostJvmAttachment) externalAttach(agentJar string, initJar string, agentHttpPort int) bool {
	initJarAbsPath, err := filepath.Abs(initJar)
	if err != nil {
		log.Error().Err(err).Msgf("Could not determine absolute path of init jar %s", initJar)
		return false
	}
	agentJarAbsPath, err := filepath.Abs(agentJar)
	if err != nil {
		log.Error().Err(err).Msgf("Could not determine absolute path of agent jar %s", agentJar)
		return false
	}
	attachCommand := []string{
		attachment.getJavaExecutable(),
		"-Xms16m",
		"-Xmx16m",
		"-XX:+UseSerialGC",
		"-XX:+PerfDisableSharedMem",
		"-Dsun.tools.attach.attachTimeout=30000",
		"-Dsteadybit.agent.disable-jvm-attachment",
		"-jar",
		initJarAbsPath,
		"pid=" + strconv.Itoa(int(attachment.Jvm.Pid)),
		"hostpid=" + strconv.Itoa(int(attachment.Jvm.Pid)),
		"host=" + attachment.GetAgentHost(),
		"port=" + strconv.Itoa(agentHttpPort),
		"agentJar=" + agentJarAbsPath,
	}

	if needUserSwitch() {
		attachCommand = addUserIdAndGroupId(attachment.Jvm, attachCommand)
	}

	log.Debug().Msgf("Executing attach command on host: %s", attachCommand)

	var ctx, cancel = context.WithTimeout(context.Background(), time.Duration(60)*time.Second)
	defer cancel()
	err = exec.CommandContext(ctx, attachCommand[0], attachCommand[1:]...).Run()
	if err != nil {
		log.Error().Err(err).Msgf("Error attaching to JVM %+v: %s", attachment.Jvm, err)
		return false
	}
	return true
}

func addUserIdAndGroupId(vm *jvm.JavaVm, attachCommand []string) []string {
	if vm.GroupId != "" && vm.UserId != "" {
		return append(attachCommand, "uid="+vm.UserId, "gid="+vm.GroupId)
	}
	return attachCommand
}

func needUserSwitch() bool {
	//TODO: implement
	return true
}

func (attachment HostJvmAttachment) getJavaExecutable() string {
	if attachment.Jvm.Path != "" && IsExecAny(attachment.Jvm.Path) {
		return attachment.Jvm.Path
	} else {
		return "java"
	}
}

func IsExecAny(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return stat.Mode()&0111 != 0
}
func (attachment HostJvmAttachment) CopyFiles(dstPath string, files map[string]string) {
	panic("not supported")
}

func (attachment HostJvmAttachment) GetAgentHost() string {
	return "127.0.0.1"
}
