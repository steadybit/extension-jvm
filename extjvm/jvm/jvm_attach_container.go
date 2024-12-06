package jvm

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

var (
	publicAddress string
)

type containerJvmAttachment struct {
	jvm JavaVmInContainer
}

func (a containerJvmAttachment) attach(agentJar string, initJar string, agentHTTPPort int) bool {
	if !a.jvm.IsRunning() {
		log.Debug().Msgf("Process not running. Skipping a to JVM %+v", a.jvm)
		return false
	}

	files := map[string]string{
		"steadybit-javaagent-main.jar": agentJar,
		"steadybit-javaagent-init.jar": initJar,
	}
	copiedFiles, err := a.copyFiles("/tmp", files)
	if err != nil {
		log.Error().Err(err).Msgf("Error copying files to container")
		return false
	}
	return externalAttach(a.jvm, copiedFiles["steadybit-javaagent-main.jar"], copiedFiles["steadybit-javaagent-init.jar"], agentHTTPPort, a.GetHostAddress(), a.jvm.PidInContainer(), a.jvm.Pid(), a.jvm.ContainerId())
}

func (a containerJvmAttachment) canAccessHostFiles() bool {
	return false
}

func (a containerJvmAttachment) copyFiles(dstPath string, files map[string]string) (map[string]string, error) {
	processRoot := filepath.Join("/proc", strconv.Itoa(int(a.jvm.Pid())), "root")
	result := make(map[string]string)

	for filename, srcFile := range files {
		destFile := filepath.Join(processRoot, dstPath, filename)
		result[filename] = filepath.Join(dstPath, filename)

		srcFileStat, err := os.Stat(srcFile)
		if err != nil {
			log.Error().Msgf("Error reading file %s: %s", srcFile, err)
			return nil, err
		}

		if destStat, _ := os.Stat(destFile); destStat != nil && srcFileStat.ModTime() == destStat.ModTime() {
			log.Trace().Msgf("File %s already exists and is up to date. Skipping copy.", destFile)
			continue
		}

		ctx := context.Background()
		if out, err := utils.RootCommandContext(ctx, "cp", srcFile, destFile).CombinedOutput(); err != nil {
			return nil, fmt.Errorf("error copying file %s: %s - %s", srcFile, err, out)
		}

		if out, err := utils.RootCommandContext(ctx, "chmod", "a+rwx", destFile).CombinedOutput(); err != nil {
			log.Warn().Msgf("error setting file permissions for %s: %s - %s", destFile, err, out)
		}

		if out, err := utils.RootCommandContext(ctx, "touch", "-m", "-d", srcFileStat.ModTime().Format("2006-01-02T15:04:05"), destFile).CombinedOutput(); err != nil {
			log.Warn().Msgf("error setting file modification time for %s: %s - %s", destFile, err, out)
		}
	}
	return result, nil
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
