package jvm

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/java_process"
	"github.com/steadybit/extension-jvm/extjvm/procfs"
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
	processRoot := procfs.GetProcessRoot(a.jvm.Pid)

	result := make(map[string]string)

	for filename, sourceFile := range files {
		destinationFile := filepath.Join(processRoot, dstPath, filename)
		result[filename] = dstPath + "/" + filename
		sourceFileStat, err := os.Stat(sourceFile)
		if err != nil {
			log.Error().Msgf("Error reading file %s: %s", sourceFile, err)
			return nil, err
		}
		destinationFileStat, _ := os.Stat(destinationFile)

		if destinationFileStat != nil && sourceFileStat.ModTime() == destinationFileStat.ModTime() {
			log.Trace().Msgf("File %s already exists and is up to date. Skipping copy.", destinationFile)
			continue
		}

		copyCommand := utils.RootCommandContext(context.Background(), "cp", sourceFile, destinationFile)
		err = copyCommand.Run()
		if err != nil {
			log.Error().Msgf("Copying file failed %s: %s", destinationFile, err)
			return nil, err
		}
		chmodCommand := utils.RootCommandContext(context.Background(), "chmod", "777", destinationFile)
		err = chmodCommand.Run()
		if err != nil {
			log.Warn().Msgf("Error setting file permissions %s: %s", destinationFile, err)
			continue
		}
		setLastModifiedCommand := utils.RootCommandContext(context.Background(), "touch", "-m", "-d", sourceFileStat.ModTime().Format("2006-01-02T15:04:05"), destinationFile)
		err = setLastModifiedCommand.Run()
		if err != nil {
			log.Warn().Msgf("Error setting file modification time %s: %s", destinationFile, err)
			continue
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
