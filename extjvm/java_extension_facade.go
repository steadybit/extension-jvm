package extjvm

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/attachment"
	"github.com/steadybit/extension-jvm/extjvm/attachment/plugin_tracking"
	"github.com/steadybit/extension-jvm/extjvm/attachment/remote_jvm_connections"
	"github.com/steadybit/extension-jvm/extjvm/common"
	"github.com/steadybit/extension-jvm/extjvm/java_process"
	"github.com/steadybit/extension-kit/extutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type JavaExtensionFacade struct{}

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
	AddListener(&JavaExtensionFacade{})
}

func worker(jobs chan AttachJvmWork) {
	for job := range jobs {
		doAttach(job)
	}

}

func doAttach(job AttachJvmWork) {
	jvm := job.jvm
	ok, err := attachInternal(jvm)
	if err != nil {
		if java_process.IsRunningProcess(jvm.Pid) {
			log.Warn().Msgf("Error attaching to JVM %+v: %s", jvm, err)
		} else {
			log.Trace().Msgf("Jvm stopped, attach failed. JVM %+v: %s", jvm, err)
		}
		return
	}
	if ok {
		log.Debug().Msgf("Successful attachment to JVM  %+v", jvm)

		loglevel := getJvmExtensionLogLevel()
		log.Trace().Msgf("Propagating Loglevel %s to Javaagent in JVM %+v", loglevel, jvm)
		if !setLogLevel(jvm, loglevel) {
			//If setting the loglevel fails, the connection has some issue - do retry
			attach(job.jvm)
		}
		for _, plugin := range autoloadPlugins {
			loadAutoLoadPlugin(jvm, plugin.MarkerClass, plugin.Plugin)
		}
	} else {
		log.Debug().Msgf("Attach to JVM skipped. Excluding %+v", jvm)
	}

}

func LoadAgentPlugin(jvm *JavaVm, plugin string, args string) (bool, error) {
	if hasAgentPlugin(jvm, plugin) {
		return true, nil
	}

	_, err := os.Stat(plugin)
	if err != nil {
		log.Error().Msgf("Plugin %s not found: %s", plugin, err)
		return false, err
	}

	var pluginPath string
	if jvm.IsRunningInContainer() {
		file := filepath.Base(plugin)
		file = fmt.Sprintf("steadybit-%s", file)
		attachment.GetAttachment(jvm).CopyFiles("/tmp", map[string]string{
			file: plugin,
		})
		pluginPath = "/tmp/" + file
	} else {
		pluginPath = plugin
	}

	loaded := SendCommandToAgent(jvm, "load-agent-plugin", fmt.Sprintf("%s,%s", pluginPath, args))
	if loaded {
		plugin_tracking.Add(jvm, plugin)
	}
	return false, nil
}

func unloadAutoLoadPlugin(jvm *JavaVm, markerClass string, plugin string) {
	if hasClassLoaded(jvm, markerClass) {
		log.Trace().Msgf("Unloading plugin %s for JVM %+v", plugin, jvm)
		_, err := UnloadAgentPlugin(jvm, plugin)
		if err != nil {
			log.Warn().Msgf("Unloading plugin %s for JVM %+v failed: %s", plugin, jvm, err)
		}
	}
}

func UnloadAgentPlugin(jvm *JavaVm, plugin string) (bool, error) {
	_, err := os.Stat(plugin)
	if err != nil {
		log.Error().Msgf("Plugin %s not found: %s", plugin, err)
		return false, err
	}

	var args string
	if jvm.IsRunningInContainer() {
		file := filepath.Base(plugin)
		file = fmt.Sprintf("steadybit-%s", file)
		args = "/tmp/" + file + ",deleteFile=true"
	} else {
		args = plugin
	}

	unloaded := SendCommandToAgent(jvm, "unload-agent-plugin", args)
	if unloaded {
		plugin_tracking.Remove(jvm, plugin)
	}
	return unloaded, nil
}
func hasAgentPlugin(jvm *JavaVm, plugin string) bool {
	return plugin_tracking.Has(jvm, plugin)
}

func hasClassLoaded(jvm *JavaVm, className string) bool {
	return SendCommandToAgent(jvm, "class-loaded", className)
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
  err = conn.SetDeadline(time.Now().Add(time.Duration(10) * time.Second))
  if err != nil {
    log.Error().Msgf("Error setting deadline for connection to JVM %d: %s", pid, err)
    return nil
  }
  err = conn.SetWriteDeadline(time.Now().Add(time.Duration(10) * time.Second))
  if err != nil {
    log.Error().Msgf("Error setting write deadline for connection to JVM %d: %s", pid, err)
    return nil
  }
  err = conn.SetReadDeadline(time.Now().Add(time.Duration(10) * time.Second))
  if err != nil {
    log.Error().Msgf("Error setting read deadline for connection to JVM %d: %s", pid, err)
    return nil
  }
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

func attachInternal(jvm *JavaVm) (bool, error) {
	if isAttached(jvm) {
		log.Trace().Msgf("RemoteJvmConnection to JVM already established. %+v", jvm)
		return true, nil
	}

	log.Debug().Msgf("RemoteJvmConnection to JVM not found. Attaching now. %+v", jvm)

	if _, err := os.Stat(JavaagentMainJar); err != nil {
		log.Error().Msgf("JavaagentMainJar not found: %s", JavaagentMainJar)
		return false, err
	}

	if _, err := os.Stat(JavaagentInitJar); err != nil {
		log.Error().Msgf("JavaagentInitJar not found: %s", JavaagentInitJar)
		return false, err
	}

	attached := attachment.GetAttachment(jvm).Attach(JavaagentMainJar, JavaagentInitJar, int(common.GetOwnJVMAttachmentPort()))
	if !attached {
		return false, errors.New("could not attach to JVM")
	}

	jvmConnectionPresent := remote_jvm_connections.WaitForConnection(jvm.Pid, time.Duration(90)*time.Second)
	if !jvmConnectionPresent {
		log.Error().Msgf("JVM with did not call back after 90 seconds.")
		return false, errors.New("could not attach to JVM: VM with did not call back after 90 seconds")
	}
	return true, nil
}

func isAttached(jvm *JavaVm) bool {
	connection := remote_jvm_connections.GetConnection(jvm.Pid)
	return connection != nil
}

func (j JavaExtensionFacade) AddedJvm(jvm *JavaVm) {
	attach(jvm)
}

func attach(jvm *JavaVm) {
	jobs <- AttachJvmWork{jvm: jvm, retries: 5}
}

func (j JavaExtensionFacade) RemovedJvm(jvm *JavaVm) {
	//TODO: implement
	//abortAttach(jvm.Pid)
	plugin_tracking.RemoveAll(jvm)
}

func AddAutoloadAgentPlugin(plugin string, markerClass string) {
	autoloadPlugins = append(autoloadPlugins, AutoloadPlugin{Plugin: plugin, MarkerClass: markerClass})
	jvms.Range(func(key, value interface{}) bool {
		jvm := value.(*JavaVm)
		loadAutoLoadPlugin(jvm, markerClass, plugin)
		return true
	})
}

func loadAutoLoadPlugin(jvm *JavaVm, markerClass string, plugin string) {
	if hasClassLoaded(jvm, markerClass) {
		log.Debug().Msgf("Marker class %s already loaded on JVM %d. Loading plugin %s", markerClass, jvm.Pid, plugin)
		_, err := LoadAgentPlugin(jvm, plugin, "")
		if err != nil {
			log.Warn().Msgf("Autoloading plugin failed %s for %s: %s", plugin, jvm.ToDebugString(), err)
			return
		}
	}
}

func RemoveAutoloadAgentPlugin(plugin string, markerClass string) {
	for i, p := range autoloadPlugins {
		if p.Plugin == plugin && p.MarkerClass == markerClass {
			autoloadPlugins = append(autoloadPlugins[:i], autoloadPlugins[i+1:]...)
			break
		}
	}
	jvms.Range(func(key, value interface{}) bool {
		jvm := value.(*JavaVm)
		unloadAutoLoadPlugin(jvm, plugin, markerClass)
		return true
	})
}
