package extjvm

import (
	"bufio"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/attachment"
	"github.com/steadybit/extension-jvm/extjvm/attachment/remote_jvm_connections"
	"github.com/steadybit/extension-jvm/extjvm/common"
	"github.com/steadybit/extension-jvm/extjvm/java_process"
	"github.com/steadybit/extension-kit/extutil"
	"net"
	"os"
	"strings"
	"time"
)

type JavaAgentFacade struct{}

type AttachJvmWork struct {
	jvm     *JavaVm
	retries int
}

type AutoloadPlugin struct {
	MarkerClass string
	Plugin      string
}

var (
	jobs            = make(chan AttachJvmWork)
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
			loadAutoloadPlugin(jvm, plugin.MarkerClass, plugin.Plugin)
		}
	} else {
		log.Debug().Msgf("Attach to JVM skipped. Excluding %+v", jvm)
	}

}

func loadAutoloadPlugin(jvm *JavaVm, markerClass string, plugin string) {
	//TODO implement
}

func setLogLevel(jvm *JavaVm, loglevel string) bool {
	return SendCommandToAgent(jvm, "log-level", loglevel)
}

func SendCommandToAgent(jvm *JavaVm, command string, args string) bool {
	success := SendCommandToAgentViaSocket(jvm, command, args, func(resultMessage string, rc int) bool {
		if rc == 0 {
			log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned %s", command, args, jvm.Pid, resultMessage)
			return extutil.ToBool(resultMessage)
		} else {
			log.Warn().Msgf("Command '%s:%s' to agent on PID %d returned error: %s", command, args, jvm.Pid, resultMessage)
			return false
		}
	})
	return success != nil && *success
}

func SendCommandToAgentViaSocket[T any](jvm *JavaVm, command string, args string, handler func(resultMessage string, rc int) T) *T {
	pid := jvm.Pid
	connection := remote_jvm_connections.GetConnection(pid)
	if connection == nil {
		log.Debug().Msgf("RemoteJvmConnection from PID %d not found. Command '%s:%s' not sent.", pid, command, args)
		return nil
	}

	d := net.Dialer{Timeout: time.Duration(10) * time.Second}
	conn, err := d.Dial("tcp", "127.0.0.1:8000")
	if err != nil {
		if java_process.IsRunningProcess(pid) {
			log.Error().Msgf("Command '%s' could not be sent over socket to %+v (%s): %s", command, jvm, connection.Address(), err)
		} else {
			log.Debug().Msgf("Process %d not running anymore. Removing connection to %+v:%s", pid, jvm, connection.Address())
		}

		return nil
	}
	conn.SetDeadline(time.Now().Add(time.Duration(10) * time.Second))
	conn.SetWriteDeadline(time.Now().Add(time.Duration(10) * time.Second))
	conn.SetReadDeadline(time.Now().Add(time.Duration(10) * time.Second))
	log.Trace().Msgf("Sending command '%s:%s' to agent on PID %d", command, args, pid)
	rc, err := conn.Write([]byte(command + ":" + args + "\n"))
	if err != nil {
		log.Error().Msgf("Error sending command '%s:%s' to JVM %d: %s", command, args, pid, err)
		return nil
	}
	message, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		log.Error().Msgf("Error reading result from JVM %d: %s", pid, err)
		return nil
	}

	return extutil.Ptr(handler(message, rc))
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

	if _, err := os.Stat(JavaagentMainJar); err != nil {
		log.Error().Msgf("JavaagentMainJar not found: %s", JavaagentMainJar)
	}

	if _, err := os.Stat(JavaagentInitJar); err != nil {
		log.Error().Msgf("JavaagentInitJar not found: %s", JavaagentInitJar)
	}

	attached := attachment.Attach(jvm, JavaagentMainJar, JavaagentInitJar, int(common.GetOwnJVMAttachmentPort()))
	if !attached {
		return false
	}

	jvmConnectionPresent := remote_jvm_connections.WaitForConnection(jvm.Pid, time.Duration(90)*time.Second)
	if !jvmConnectionPresent {
		log.Error().Msgf("JVM with did not call back after 90 seconds.")
		return false
	}
	return true
}

func isAttached(jvm *JavaVm) bool {
	connection := remote_jvm_connections.GetConnection(jvm.Pid)
	return connection != nil
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

// TODO call by Spring and Datanbase Discovery
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
