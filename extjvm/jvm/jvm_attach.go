package jvm

import (
	"bytes"
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/container"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

type JvmAttachment interface {
	attach(agentJar string, initJar string, port int) bool
	copyFiles(dstPath string, files map[string]string) (map[string]string, error)
	GetHostAddress() string
}

func GetAttachment(jvm *JavaVm) JvmAttachment {
	if jvm.IsRunningInContainer() {
		return containerJvmAttachment{
			jvm: jvm,
		}
	}
	return hostJvmAttachment{
		jvm: jvm,
	}
}

func externalAttach(jvm *JavaVm, agentJar string, initJar string, agentHTTPPort int, host string, addRuncExec bool, pid string, hostpid string) bool {
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
		getJavaExecutable(jvm),
		"-Xms16m",
		"-Xmx16m",
		"-XX:+UseSerialGC",
		"-XX:+PerfDisableSharedMem",
		"-Dsun.tools.attach.attachTimeout=30000",
		"-Dsteadybit.agent.disable-jvm-attachment",
		"-jar",
		initJarAbsPath,
		"pid=" + pid,
		"hostpid=" + hostpid,
		"host=" + host,
		"port=" + strconv.Itoa(agentHTTPPort),
		"agentJar=" + agentJarAbsPath,
	}

	if addRuncExec {
		// We first enter the init process of the container and then execute the runc exec command because otherwise we fail with "exec failed: container_linux.go:367: starting container process caused: process_linux.go:96: starting setns process caused: fork/ │
		//│ exec /proc/self/exe: no such file or directory"  "
		runcExecCommand := []string{"nsenter", "-t", "1", "-m", "-n", "-i", "-p", "-r", "-u", "-C", "--", "runc", "--root", container.GetRuncRoot(), "exec", jvm.ContainerId}
		attachCommand = append(runcExecCommand, attachCommand...)
	}

	if needsUserSwitch(jvm) {
		attachCommand = addUserIdAndGroupId(jvm, attachCommand)
	}

	log.Debug().Msgf("Executing attach command on host: %s", attachCommand)

	var ctx, cancel = context.WithTimeout(context.Background(), time.Duration(60)*time.Second)
	defer cancel()
	cmd := utils.RootCommandContext(ctx, attachCommand[0], attachCommand[1:]...)
	var outb, errb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	err = cmd.Run()
	log.Debug().Msgf("attach command output: %s", outb.String())
	if errb.String() != "" {
		log.Error().Msgf("attach command error: %s", errb.String())
	}
	if err != nil {
		log.Error().Err(err).Msgf("Error attaching to JVM %+v: %s", jvm, err)
		return false
	}
	return true
}

func addUserIdAndGroupId(vm *JavaVm, attachCommand []string) []string {
	if vm.GroupId != "" && vm.UserId != "" {
		return append(attachCommand, "uid="+vm.UserId, "gid="+vm.GroupId)
	}
	return attachCommand
}

func needsUserSwitch(jvm *JavaVm) bool {
	current, err := user.Current()
	if err != nil {
		log.Warn().Err(err).Msg("Could not determine current user")
		return false
	}
	return !(jvm.UserId == current.Uid && jvm.GroupId == current.Gid)
}

func getJavaExecutable(jvm *JavaVm) string {
	if jvm.ContainerId != "" {
		return jvm.Path
	}
	if jvm.Path != "" && (isExecAny(jvm.Path)) {
		return jvm.Path
	} else {
		if runtime.GOOS == "windows" {
			return "java.exe"
		}
		return "java"
	}
}

func isExecAny(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}
	return stat.Mode()&0111 != 0
}
