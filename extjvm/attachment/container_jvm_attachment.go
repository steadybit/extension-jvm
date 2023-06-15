package attachment

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/java_process"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-jvm/extjvm/procfs"
	"github.com/steadybit/extension-jvm/extjvm/utils"
  "net"
  "os"
	"path/filepath"
  "strconv"
)

var (
  PublicAddress string
)

type ContainerJvmAttachment struct {
	Jvm *jvm.JavaVm
}

func (attachment ContainerJvmAttachment) Attach(agentJar string, initJar string, agentHTTPPort int) bool {
	if !java_process.IsRunningProcess(attachment.Jvm.Pid) {
		log.Debug().Msgf("Process not running. Skipping attachment to JVM %+v", attachment.Jvm)
		return false
	}

  files := map[string]string{
    "steadybit-javaagent-main.jar": agentJar,
    "steadybit-javaagent-init.jar": initJar,
  }
  copiedFiles, err := attachment.CopyFiles("/tmp", files)
  if err != nil {
    log.Error().Err(err).Msgf("Error copying files to container")
    return false
  }
	return externalAttach(attachment.Jvm, copiedFiles["steadybit-javaagent-main.jar"], copiedFiles["steadybit-javaagent-init.jar"], agentHTTPPort, attachment.GetAgentHost(), true, strconv.Itoa(attachment.Jvm.InContainerPid), strconv.Itoa(int(attachment.Jvm.Pid)))
}

func (attachment ContainerJvmAttachment) CopyFiles(dstPath string, files map[string]string) (map[string]string, error) {
	processRoot := procfs.GetProcessRoot(attachment.Jvm.Pid)

  result := make(map[string]string)

	for filename, sourceFile := range files {
		destinationFile := filepath.Join(processRoot, dstPath, filename)
    result[filename] = dstPath+ "/" + filename
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

func (attachment ContainerJvmAttachment) GetAgentHost() string {
  if PublicAddress != "" {
    return PublicAddress
  }
  address := os.Getenv("POD_IP")
  if address != "" {
    PublicAddress = address
    return address
  }
  address = os.Getenv("STEADYBIT_EXTENSION_CONTAINER_ADDRESS")
  if address != "" {
    PublicAddress = address
    return address
  }

  ip := getOutboundIP()
  if ip != nil {
    PublicAddress = ip.String()
    return ip.String()
  }
	return PublicAddress
}

// Get preferred outbound ip of this machine
func getOutboundIP() net.IP {
  conn, err := net.Dial("udp", "8.8.8.8:80")
  if err != nil {
    log.Error().Err(err).Msgf("Error getting outbound IP")
  }
  defer conn.Close()

  localAddr := conn.LocalAddr().(*net.UDPAddr)

  return localAddr.IP
}
