package jvm

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/java_process"
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
	jvm *JavaVm
}

func (a containerJvmAttachment) attach(agentJar string, initJar string, agentHTTPPort int) bool {
	if !java_process.IsRunningProcess(a.jvm.Pid) {
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
	return externalAttach(a.jvm, copiedFiles["steadybit-javaagent-main.jar"], copiedFiles["steadybit-javaagent-init.jar"], agentHTTPPort, a.GetHostAddress(), true, strconv.Itoa(a.jvm.InContainerPid), strconv.Itoa(int(a.jvm.Pid)))
}

func (a containerJvmAttachment) copyFiles(dstPath string, files map[string]string) (map[string]string, error) {
	processRoot := filepath.Join("/proc", strconv.Itoa(int(a.jvm.Pid)), "root")
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

		copyCmd := utils.RootCommandContext(context.Background(), "cp", srcFile, destFile)
		if out, err := copyCmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("error copying file %s: %s - %s", srcFile, err, out)
		}

		chmodCmd := utils.RootCommandContext(context.Background(), "chmod", "777", destFile)
		if out, err := chmodCmd.CombinedOutput(); err != nil {
			log.Warn().Msgf("error setting file permissions for %s: %s - %s", destFile, err, out)
		}

		setLastModifiedMd := utils.RootCommandContext(context.Background(), "touch", "-m", "-d", srcFileStat.ModTime().Format("2006-01-02T15:04:05"), destFile)
		if out, err := setLastModifiedMd.CombinedOutput(); err != nil {
			log.Warn().Msgf("error setting file modification time for %s: %s - %s", destFile, err, out)
		}
	}
	return result, nil
}

func (a containerJvmAttachment) GetHostAddress() string {
	if publicAddress != "" {
		return publicAddress
	}
	address := os.Getenv("POD_IP")
	if address != "" {
		publicAddress = address
		return address
	}
	address = os.Getenv("STEADYBIT_EXTENSION_CONTAINER_ADDRESS")
	if address != "" {
		publicAddress = address
		return address
	}

	ip := getOutboundIP()
	if ip != nil {
		publicAddress = ip.String()
		return ip.String()
	}
	return publicAddress
}

// Get preferred outbound ip of this machine
func getOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Error().Err(err).Msgf("Error getting outbound IP")
	}
	defer func() { _ = conn.Close() }()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP
}
