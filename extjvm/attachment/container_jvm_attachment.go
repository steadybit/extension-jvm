package attachment

import (
	"context"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/java_process"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-jvm/extjvm/procfs"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"os"
	"path/filepath"
)

type ContainerJvmAttachment struct {
	Jvm *jvm.JavaVm
}

func (attachment ContainerJvmAttachment) Attach(agentJar string, initJar string, agentHTTPPort int) bool {
	if !java_process.IsRunningProcess(attachment.Jvm.Pid) {
		log.Debug().Msgf("Process not running. Skipping attachment to JVM %+v", attachment.Jvm)
		return false
	}
	return externalAttach(attachment.Jvm, agentJar, initJar, agentHTTPPort, attachment.GetAgentHost(), true)
}

func (attachment ContainerJvmAttachment) CopyFiles(dstPath string, files map[string]string) {
	processRoot := procfs.GetProcessRoot(attachment.Jvm.Pid)
	for filename, sourceFile := range files {
		destinationFile := filepath.Join(processRoot, dstPath, filename)

		sourceFileStat, err := os.Stat(sourceFile)
		if err != nil {
			log.Error().Msgf("Error reading file %s: %s", sourceFile, err)
			continue
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
			continue
		}
		chmodCommand := utils.RootCommandContext(context.Background(), "chmod", "777", destinationFile)
		err = chmodCommand.Run()
		if err != nil {
			log.Error().Msgf("Error setting file permissions %s: %s", destinationFile, err)
			continue
		}
		setLastModifiedCommand := utils.RootCommandContext(context.Background(), "touch", "-m", "-d", sourceFileStat.ModTime().Format("2006-01-02T15:04:05"), destinationFile)
		err = setLastModifiedCommand.Run()
		if err != nil {
			log.Error().Msgf("Error setting file modification time %s: %s", destinationFile, err)
			continue
		}
	}
}

func (attachment ContainerJvmAttachment) GetAgentHost() string {
	return "127.0.0.1"
}
