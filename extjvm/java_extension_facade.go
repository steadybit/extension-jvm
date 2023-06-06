package extjvm

import (
  "github.com/rs/zerolog/log"
  "github.com/steadybit/extension-jvm/extjvm/attachment"
  "github.com/steadybit/extension-jvm/extjvm/common"
  "os"
	"strings"
  "time"
)

type JavaAgentFacade struct{}

type AttachJvmWork struct {
  jvm *JavaVm
  retries int
}

type AutoloadPlugin struct {
  MarkerClass string
  Plugin      string
}

var (
  jobs = make(chan AttachJvmWork)
  autoloadPlugins = make([]AutoloadPlugin, 0)

  JavaagentInitJar = "/javaagent/javaagent-init.jar"
  JavaagentMainJar = "/javaagent/javaagent-main.jar"
)

func StartAttachment() {
	attachmentEnabled := os.Getenv("STEADYBIT_EXTENSION_JVM_ATTACHMENT_ENABLED")
	if strings.ToLower(attachmentEnabled) != "true" {
		return
	}
  // create worker pool
  for w := 1; w <= 4; w++ {
    go worker(jobs)
  }
	AddListener(&JavaAgentFacade{})
}

func worker(jobs chan AttachJvmWork) {
  for job := range jobs {
    doAttach(job)
  }

}

func doAttach(job AttachJvmWork) {
  jvm := job.jvm
  if attachInternal(jvm) {
    log.Debug().Msgf("Successful attachment to JVM  %+v", jvm)

    loglevel := getJvmExtensionLogLevel()
    log.Trace().Msgf("Propagating Loglevel %s to Javaagent in JVM %+v", loglevel, jvm)
    if !setLogLevel(jvm, loglevel) {
      //If setting the loglevel fails, the connection has some issue - do retry
      attach(job.jvm)
    }
    for _, plugin := range autoloadPlugins {
      loadAutoloadPlugin(jvm, plugin.MarkerClass, plugin.Plugin )
    }
  } else {
    log.Debug().Msgf("Attach to JVM skipped. Excluding %+v", jvm)
  }

}

func loadAutoloadPlugin(jvm *JavaVm, markerClass string, plugin string) {
  //TODO implement
}

func setLogLevel(jvm *JavaVm, loglevel string) bool {
  //TODO implement
  return true
}

func getJvmExtensionLogLevel() string {
  loglevel := os.Getenv("STEADYBIT_EXTENSION_LOG_LEVEL")
  if loglevel == "" {
    loglevel = "info"
  }
  return strings.ToUpper(loglevel)
}

func attachInternal(jvm *JavaVm) bool {
  if isAttached(jvm) {
    log.Trace().Msgf("RemoteJvmConnection to JVM already established. %+v", jvm)
    return true
  }

  log.Debug().Msgf("RemoteJvmConnection to JVM not found. Attaching now. %+v", jvm)
  attached := attachment.Attach(jvm, JavaagentMainJar, JavaagentInitJar, int(common.GetOwnJVMAttachmentPort()))
  if !attached {
    return false
  }
  jvmConnectionPresent := attachment.WaitForConnection(jvm.Pid, time.Duration(90)*time.Second)
  if !jvmConnectionPresent {
    log.Error().Msgf("JVM with did not call back after 90 seconds.")
    return false
  }
  return true
}

func isAttached(jvm *JavaVm) bool {
  //TODO implement
  return false
}

func (j JavaAgentFacade) AddedJvm(jvm *JavaVm) {
	attach(jvm)
}

func attach(jvm *JavaVm) {
  jobs <- AttachJvmWork{jvm: jvm, retries: 5}
}

func (j JavaAgentFacade) RemovedJvm(jvm *JavaVm) {
	//TODO: implement
	//abortAttach(jvm.Pid)
	//pluginTracking.removeAll(jvm);
}

//TODO call by Spring and Datanbase Discovery
func addAutoloadAgentPlugin(plugin AutoloadPlugin) {
  autoloadPlugins = append(autoloadPlugins, plugin)
}

func removeAutoloadAgentPlugin(plugin AutoloadPlugin) {
  for i, p := range autoloadPlugins {
    if p == plugin {
      autoloadPlugins = append(autoloadPlugins[:i], autoloadPlugins[i+1:]...)
      break
    }
  }
}
