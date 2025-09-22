package jvm

import (
	"context"
	"path"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/utils"
)

type hostJvmAttachment struct {
	jvm JavaVm
}

func (a hostJvmAttachment) attach(agentHTTPPort int, heartbeatFile string) bool {
	if !a.jvm.IsRunning() {
		log.Debug().Msgf("Process not running. Skipping a to JVM %s", a.jvm.ToDebugString())
		return false
	}

	return externalAttach(a.jvm,
		a.resolveFile(mainJarName),
		a.resolveFile(initJarName),
		a.resolveFile(heartbeatFile),
		agentHTTPPort,
		a.GetHostAddress(),
		a.jvm.Pid(),
		a.jvm.Pid(),
		a.run,
	)
}

func (a hostJvmAttachment) run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return utils.RootCommandContext(ctx, name, args...).CombinedOutput()
}

func (a hostJvmAttachment) resolveFile(f string) string {
	return path.Join(javaagentPath(), f)
}

func (a hostJvmAttachment) GetHostAddress() string {
	return "127.0.0.1"
}
