package jvm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/container"
	"github.com/steadybit/extension-jvm/extjvm/utils"
)

var (
	publicAddress string
)

const (
	javaagentPathInContainer = "/tmp/.steadybit"
)

type containerJvmAttachment struct {
	jvm JavaVmInContainer
}

func (a containerJvmAttachment) attach(agentHTTPPort int, heartbeatFile string) bool {
	if !a.jvm.IsRunning() {
		log.Debug().Msgf("Process not running. Skipping a to JVM %s", a.jvm.ToDebugString())
		return false
	}

	resolvedMainJar, err1 := a.resolveFile(mainJarName)
	resolvedInitJar, err2 := a.resolveFile(initJarName)
	if err := errors.Join(err1, err2); err != nil {
		log.Error().Err(err).Msgf("failed resolving agent jars in container")
		return false
	}
	return externalAttach(a.jvm,
		resolvedMainJar,
		resolvedInitJar,
		"", // we need to copy files, so heartbeat is not supported here
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
	return path.Join(javaagentPathInContainer, f), a.copyFile(f)
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

func (a containerJvmAttachment) copyFile(f string) error {
	srcFile := filepath.Join(javaagentPath(), f)
	dstFile := filepath.Join("/proc", strconv.Itoa(int(a.jvm.Pid())), "root", javaagentPathInContainer, f)

	srcFileStat, err := os.Stat(srcFile)
	if err != nil {
		log.Error().Msgf("Error reading file %s: %s", srcFile, err)
		return err
	}

	if destStat, _ := os.Stat(dstFile); destStat != nil && srcFileStat.ModTime() == destStat.ModTime() {
		log.Trace().Msgf("File %s already exists and is up to date. Skipping copy.", dstFile)
		return nil
	}

	ctx := context.Background()
	if out, err := utils.RootCommandContext(ctx, "mkdir", "-p", path.Dir(dstFile)).CombinedOutput(); err != nil {
		log.Warn().Err(err).Str("out", string(out)).Msgf("Error creating directory %s", path.Base(dstFile))
	}

	if out, err := utils.RootCommandContext(ctx, "cp", srcFile, dstFile).CombinedOutput(); err != nil {
		return fmt.Errorf("error copying file %s: %s - %s", srcFile, err, out)
	}

	if out, err := utils.RootCommandContext(ctx, "chmod", "a+rwx", dstFile).CombinedOutput(); err != nil {
		log.Warn().Err(err).Str("out", string(out)).Msgf("error setting file permissions for %s", dstFile)
	}

	if out, err := utils.RootCommandContext(ctx, "touch", "-m", "-d", srcFileStat.ModTime().Format("2006-01-02T15:04:05"), dstFile).CombinedOutput(); err != nil {
		log.Warn().Err(err).Str("out", string(out)).Msgf("error setting file modification time for %s", dstFile)
	}

	return nil
}
