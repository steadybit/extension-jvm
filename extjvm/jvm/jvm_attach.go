package jvm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"time"

	"github.com/rs/zerolog/log"
)

type Attachment interface {
	attach(port int, heartbeatFile string) bool
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

type cmdSupplier func(ctx context.Context, name string, args ...string) *exec.Cmd

func externalAttach(jvm JavaVm, agentJar, initJar string, heartbeatFile string, agentHTTPPort int, host string, pid, hostpid int32, cmdFn cmdSupplier) bool {
	attachCommand := []string{
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
		fmt.Sprintf("host=%s", host),
		fmt.Sprintf("port=%d", agentHTTPPort),
		fmt.Sprintf("agentJar=%s", agentJar),
		fmt.Sprintf("heartbeat=%s", heartbeatFile),
	}

	if needsUserSwitch(jvm) {
		attachCommand = append(attachCommand, "uid="+jvm.UserId(), "gid="+jvm.GroupId())
	}

	var ctx, cancel = context.WithTimeout(context.Background(), time.Duration(60)*time.Second)
	defer cancel()

	cmd := cmdFn(ctx, getExecutable(jvm), attachCommand...)
	log.Debug().Strs("args", cmd.Args).Msg("executing attach command")
	outb, err := cmd.CombinedOutput()

	if err != nil {
		log.Error().Err(err).Str("output", string(outb)).Msgf("Error attaching to JVM %s: %s", jvm.ToInfoString(), err)
		return false
	}

	log.Debug().Msgf("attach command output: %s", string(outb))
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
