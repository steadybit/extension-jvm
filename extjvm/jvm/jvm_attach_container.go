package jvm

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/container"
	"github.com/steadybit/extension-jvm/extjvm/utils"
)

var (
	publicAddress string
	nsmountPath   = initNsMountPath()
)

const (
	javaagentPathInContainer = "/tmp/.steadybit"
)

func initNsMountPath() string {
	p := "nsmount"
	if fromEnv := os.Getenv("STEADYBIT_EXTENSION_NSMOUNT_PATH"); fromEnv != "" {
		p = fromEnv
	}

	if lookupPath, err := exec.LookPath(p); err == nil {
		return lookupPath
	} else {
		return p
	}
}

type containerJvmAttachment struct {
	jvm JavaVmInContainer
}

func (a containerJvmAttachment) attach(agentHTTPPort int, heartbeatFile string) bool {
	if !a.jvm.IsRunning() {
		log.Debug().Msgf("Process not running. Skipping a to JVM %s", a.jvm.ToDebugString())
		return false
	}

	if err := a.mountDirectory(javaagentPath(), javaagentPathInContainer); err != nil {
		log.Error().Err(err).Msgf("Error mounting files to container")
		return false
	}

	resolvedMainJar, err := a.resolveFile(mainJarName)
	if err != nil {
		log.Error().Err(err).Str("file", mainJarName).Msgf("failed to resolve path on host")
	}
	resolvedInitJar, err := a.resolveFile(initJarName)
	if err != nil {
		log.Error().Err(err).Str("file", initJarName).Msgf("failed to resolve path on host")
	}
	resolvedHeartbeatFile, err := a.resolveFile(heartbeatFile)
	if err != nil {
		log.Error().Err(err).Str("file", heartbeatFile).Msgf("failed to resolve path on host")
	}
	return externalAttach(a.jvm,
		resolvedMainJar,
		resolvedInitJar,
		resolvedHeartbeatFile,
		agentHTTPPort,
		a.GetHostAddress(),
		a.jvm.PidInContainer(),
		a.jvm.Pid(),
		a.run,
	)
}

func (a containerJvmAttachment) run(ctx context.Context, name string, args ...string) *exec.Cmd {
	return container.Exec(ctx, a.jvm.ContainerId(), name, args...)
}

func (a containerJvmAttachment) resolveFile(f string) (string, error) {
	return path.Join(javaagentPathInContainer, f), nil
}
func (a containerJvmAttachment) GetHostAddress() string {
	if publicAddress == "" {
		if address := os.Getenv("POD_IP"); address != "" {
			publicAddress = address
		} else if address = os.Getenv("STEADYBIT_EXTENSION_CONTAINER_ADDRESS"); address != "" {
			publicAddress = address
		} else if ip := getOutboundIP(); ip != nil {
			publicAddress = ip.String()
		}
	}
	return publicAddress
}

// getOutboundIP returns preferred outbound ip of this machine
func getOutboundIP() net.IP {
	if conn, err := net.Dial("udp", "google.com:80"); err != nil {
		log.Error().Err(err).Msgf("Error getting outbound IP")
		return nil
	} else {
		defer func() { _ = conn.Close() }()
		return conn.LocalAddr().(*net.UDPAddr).IP
	}
}

func (a containerJvmAttachment) mountDirectory(srcPath, dstPath string) error {
	jvmPid := strconv.Itoa(int(a.jvm.Pid()))
	fullDestPath := filepath.Join("/proc", jvmPid, "root", dstPath)

	if out, err := utils.RootCommandContext(context.Background(), "rmdir", fullDestPath).CombinedOutput(); err != nil {
		if !strings.Contains(string(out), "No such file or directory") {
			log.Debug().Err(err).Bytes("out", out).Msgf("error removing path %s", fullDestPath)
		}
	}

	if out, err := utils.RootCommandContext(context.Background(), "mkdir", "-p", fullDestPath).CombinedOutput(); err != nil {
		return fmt.Errorf("error creating path %s: %w - %s", fullDestPath, err, out)
	}

	if out, err := utils.RootCommandContext(context.Background(), nsmountPath, strconv.Itoa(os.Getpid()), srcPath, jvmPid, dstPath).CombinedOutput(); err != nil {
		return fmt.Errorf("error mounting %s to %s for pid %d: %w - %s", srcPath, dstPath, a.jvm.Pid(), err, out)
	}

	log.Debug().Str("srcPath", srcPath).Str("dstPath", dstPath).Str("containerId", a.jvm.ContainerId()).Msg("mounted directory in container")
	return nil
}
