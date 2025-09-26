package jvm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
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

var (
	javaagentWorkDir     string
	javaagentWorkDirOnce = sync.OnceFunc(func() {
		paths := strings.SplitN(os.Getenv("STEADYBIT_EXTENSION_JAVA_AGENT_PATH"), ":", 2)
		if len(paths) == 0 {
			panic("STEADYBIT_EXTENSION_JAVA_AGENT_PATH not set")
		}

		if len(paths) == 1 {
			javaagentWorkDir = paths[0]
			return
		}

		// A single time we copy the javaagent files to a writable location, as there will be also the heartbeat file created
		javaagentWorkDir = paths[1]
		if err := os.MkdirAll(javaagentWorkDir, 0777); err != nil {
			panic("Could not create javaagent working directory: " + err.Error())
		}

		if err := os.CopyFS(javaagentWorkDir, os.DirFS(paths[0])); err != nil {
			panic("Could not copy javaagent: " + err.Error())
		}

		log.Info().Str("dir", javaagentWorkDir).Msg("prepared javaagent directory")
	})
)

func javaagentPath() string {
	javaagentWorkDirOnce()
	return javaagentWorkDir
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
	}

	if heartbeatFile != "" {
		attachCommand = append(attachCommand, fmt.Sprintf("heartbeatFile=%s", heartbeatFile))
	}

	if needsUserSwitch(jvm) {
		attachCommand = append(attachCommand, fmt.Sprintf("uid=%d", jvm.UserId()), fmt.Sprintf("gid=%d", jvm.GroupId()))
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
	if jvm.UserId() == -1 || jvm.GroupId() == -1 {
		return false
	}

	return jvm.UserId() != os.Getuid() || jvm.GroupId() != os.Getgid()
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
