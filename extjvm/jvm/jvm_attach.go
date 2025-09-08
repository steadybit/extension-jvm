package jvm

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/container"
	"github.com/steadybit/extension-jvm/extjvm/utils"
)

type Attachment interface {
	attach(port int) bool
	resolveFile(f string) string
	GetHostAddress() string
}

const (
	mainJarName = "javaagent-main.jar"
	initJarName = "javaagent-init.jar"
)

func javaagentPath() string {
	pathByEnv := os.Getenv("STEADYBIT_EXTENSION_JAVA_AGENT_PATH")
	if pathByEnv != "" {
		return pathByEnv
	}
	panic("STEADYBIT_EXTENSION_JAVA_AGENT_PATH not set")
}

func GetAttachment(jvm JavaVm) Attachment {
	if c, ok := jvm.(JavaVmInContainer); ok {
		return containerJvmAttachment{
			jvm: c,
		}
	}
	return hostJvmAttachment{
		jvm: jvm,
	}
}

func externalAttach(jvm JavaVm, agentJar string, initJar string, agentHTTPPort int, host string, pid int32, hostpid int32, containerId string) bool {
	attachCommand := []string{
		getExecutable(jvm),
		"-Xms16m",
		"-Xmx16m",
		"-XX:+UseSerialGC",
		"-XX:+PerfDisableSharedMem",
		"-Dsun.tools.attach.attachTimeout=30000",
		"-Dsteadybit.agent.disable-jvm-attachment",
		"-jar",
		initJar,
		fmt.Sprintf("pid=%d", pid),
		fmt.Sprintf("hostpid=%d", hostpid),
		"host=" + host,
		fmt.Sprintf("port=%d", agentHTTPPort),
		"agentJar=" + agentJar,
	}

	if containerId != "" {
		// We first enter the init process of the host and then execute the runc exec command because otherwise we fail with "exec failed: container_linux.go:367: starting container process caused: process_linux.go:96: starting setns process caused: fork/ │
		//│ exec /proc/self/exe: no such file or directory"  "
		runcExecCommand := []string{"nsenter", "-t", "1", "-m", "-n", "-i", "-p", "-r", "-u", "-C", "--", "runc", "--root", container.GetRuncRoot(), "exec", containerId}
		attachCommand = append(runcExecCommand, attachCommand...)
	}

	if needsUserSwitch(jvm) {
		attachCommand = append(attachCommand, "uid="+jvm.UserId(), "gid="+jvm.GroupId())
	}

	log.Debug().Msgf("Executing attach command on host: %s", attachCommand)
	var ctx, cancel = context.WithTimeout(context.Background(), time.Duration(60)*time.Second)
	defer cancel()
	outb, err := utils.RootCommandContext(ctx, attachCommand[0], attachCommand[1:]...).CombinedOutput()
	log.Debug().Msgf("attach command output: %s", string(outb))
	if err != nil {
		log.Error().Err(err).Str("output", string(outb)).Msgf("Error attaching to JVM %s: %s", jvm.ToInfoString(), err)
		return false
	}
	return true
}

func needsUserSwitch(jvm JavaVm) bool {
	if jvm.UserId() == "" || jvm.GroupId() == "" {
		return false
	}

	current, err := user.Current()
	if err != nil {
		log.Warn().Err(err).Msg("Could not determine current user")
		return false
	}
	return jvm.UserId() != current.Uid || jvm.GroupId() != current.Gid
}

func getExecutable(jvm JavaVm) string {
	if _, ok := jvm.(JavaVmInContainer); ok {
		return jvm.Path()
	}
	if jvm.Path() != "" && (isExecAny(jvm.Path())) {
		return jvm.Path()
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
