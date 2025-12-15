package jvm

import (
	"context"
	"os/exec"
	"path"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/utils"
)

type hostJvmAttachment struct {
	jvm JavaVm
}

func (a hostJvmAttachment) attach(agentHTTPPort int, heartbeatFile string) bool {
	if !a.jvm.IsRunning() {
		log.Debug().Msgf("Process not running. Skipping JVM %s", a.jvm.ToDebugString())
		return false
	}

	resolvedMainJar, err := a.resolveFile(mainJarName)
	if err != nil {
		log.Error().Err(err).Str("file", mainJarName).Msgf("failed to resolve path in container")
	}
	resolvedInitJar, err := a.resolveFile(initJarName)
	if err != nil {
		log.Error().Err(err).Str("file", initJarName).Msgf("failed to resolve path in container")
	}
	resolvedHeartbeatFile, err := a.resolveFile(heartbeatFile)
	if err != nil {
		log.Error().Err(err).Str("file", heartbeatFile).Msgf("failed to resolve path in container")
	}
	return externalAttach(a.jvm,
		resolvedMainJar,
		resolvedInitJar,
		resolvedHeartbeatFile,
		agentHTTPPort,
		a.GetHostAddress(),
		a.jvm.Pid(),
		a.jvm.Pid(),
		a.run,
	)
}

func (a hostJvmAttachment) run(ctx context.Context, name string, args ...string) *exec.Cmd {
	return utils.RootCommandContext(ctx, name, args...)
}

func (a hostJvmAttachment) resolveFile(f string) (string, error) {
	return path.Join(javaagentPath(), f), nil
}

func (a hostJvmAttachment) GetHostAddress() string {
	return "127.0.0.1"
}
