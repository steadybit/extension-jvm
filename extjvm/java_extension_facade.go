package extjvm

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/dimchansky/utfbom"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/attachment"
	"github.com/steadybit/extension-jvm/extjvm/attachment/plugin_tracking"
	"github.com/steadybit/extension-jvm/extjvm/attachment/remote_jvm_connections"
	"github.com/steadybit/extension-jvm/extjvm/common"
	"github.com/steadybit/extension-jvm/extjvm/java_process"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-kit/extutil"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type JavaExtensionFacade struct{}

type AttachJvmWork struct {
	jvm     *jvm.JavaVm
	retries int
}

type LoadPluginJvmWork struct {
	jvm     *jvm.JavaVm
	plugin  string
	args    string
	retries int
}

type AutoloadPlugin struct {
	MarkerClass string
	Plugin      string
}

type AttachedListener interface {
	JvmAttachedSuccessfully(jvm *jvm.JavaVm)
	AttachedProcessStopped(jvm *jvm.JavaVm)
}

var (
	attachJobs      = make(chan AttachJvmWork)
	loadPluginJobs  = make(chan LoadPluginJvmWork)
	autoloadPlugins = make([]AutoloadPlugin, 0)

	autoloadPluginsMutex sync.Mutex

	JavaagentInitJar = common.GetJarPath("javaagent-init.jar")
	JavaagentMainJar = common.GetJarPath("javaagent-main.jar")

	attachedListeners []AttachedListener

	SocketTimeout = 10 * time.Second
)

func StartAttachment() {
	attachmentEnabled := os.Getenv("STEADYBIT_EXTENSION_JVM_ATTACHMENT_ENABLED")
	if attachmentEnabled != "" && strings.ToLower(attachmentEnabled) != "true" {
		return
	}
	// create attach worker pool
	for w := 1; w <= 4; w++ {
		go attachWorker(attachJobs)
	}
	// create plugin load worker pool
	for w := 1; w <= 4; w++ {
		go loadPluginWorker(loadPluginJobs)
	}
	AddListener(&JavaExtensionFacade{})
}

func AddAttachedListener(attachedListener AttachedListener) {
	attachedListeners = append(attachedListeners, attachedListener)
	for _, jvm := range GetJVMs() {
		attachedListener.JvmAttachedSuccessfully(&jvm)
	}
}

func attachWorker(attachJobs chan AttachJvmWork) {
	for job := range attachJobs {
		job.retries--
		if job.retries > 0 {
			doAttach(job)
		} else {
			log.Warn().Msgf("Attach retries for %s exceeded.", job.jvm.ToDebugString())
		}
	}
}
func loadPluginWorker(loadPluginJobs chan LoadPluginJvmWork) {
	for job := range loadPluginJobs {
		job.retries--
		if job.retries > 0 {
			loadAgentPluginJob(job)
		} else {
			log.Warn().Msgf("Load Plugin retries for %s with plugin %s exceeded.", job.jvm.MainClass, job.plugin)
		}
	}
}

func doAttach(job AttachJvmWork) {
	jvm := job.jvm
	ok, err := attachInternal(jvm)
	if err != nil {
		if java_process.IsRunningProcess(jvm.Pid) {
			log.Warn().Msgf("Error attaching to JVM %+v: %s", jvm, err)
			go func() {
				time.Sleep(10 * time.Second)
				// do retry
				attach(job.jvm, job.retries)
			}()

		} else {
			log.Trace().Msgf("Jvm stopped, attach failed. JVM %+v: %s", jvm, err)
		}
		return
	}
	if ok {
		log.Info().Msgf("Successful attachment to JVM  %+v", jvm)

		loglevel := getJvmExtensionLogLevel()
		log.Trace().Msgf("Propagating Loglevel %s to Javaagent in JVM %+v", loglevel, jvm)
		if !setLogLevel(jvm, loglevel) {
			//If setting the loglevel fails, the connection has some issue - do retry
			attach(job.jvm, job.retries)
		} else {
			log.Info().Msgf("Successfully set loglevel %s for JVM %+v", loglevel, jvm)
		}
		for _, plugin := range autoloadPlugins {
			loadAutoLoadPlugin(jvm, plugin.MarkerClass, plugin.Plugin)
		}
		informListeners(jvm)
	} else {
		log.Debug().Msgf("Attach to JVM skipped. Excluding %+v", jvm)
	}

}

func informListeners(vm *jvm.JavaVm) {
	for _, listener := range attachedListeners {
		listener := listener
		go func() {
			listener.JvmAttachedSuccessfully(vm)
		}()
	}
}

func scheduleLoadAgentPlugin(jvm *jvm.JavaVm, plugin string, args string, retries int) {
	loadPluginJobs <- LoadPluginJvmWork{
		jvm:     jvm,
		plugin:  plugin,
		args:    args,
		retries: retries,
	}
}

func loadAgentPluginJob(job LoadPluginJvmWork) {
	success, err := LoadAgentPlugin(job.jvm, job.plugin, job.args)
	if err != nil || !success {
		log.Error().Msgf("Error loading plugin %s for JVM %+v: %s", job.plugin, job.jvm, err)
		go func() {
			time.Sleep(10 * time.Second)
			// do retry
			scheduleLoadAgentPlugin(job.jvm, job.plugin, job.args, job.retries)
		}()
		return
	}
}

func LoadAgentPlugin(jvm *jvm.JavaVm, plugin string, args string) (bool, error) {
	if HasAgentPlugin(jvm, plugin) {
		log.Info().Msgf("Plugin %s already loaded for JVM %+v", plugin, jvm)
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
		files, err := attachment.GetAttachment(jvm).CopyFiles("/tmp", map[string]string{
			file: plugin,
		})
		if err != nil {
			log.Error().Msgf("Error copying plugin %s to container: %s", plugin, err)
			return false, err
		}
		pluginPath = files[file]
	} else {
		pluginPath = plugin
	}

	loaded := sendCommandToAgent(jvm, "load-agent-plugin", fmt.Sprintf("%s,%s", pluginPath, args), time.Duration(30)*time.Second)
	if loaded {
		log.Info().Msgf("Plugin %s loaded for JVM %+v", plugin, jvm)
		plugin_tracking.Add(jvm.Pid, plugin)
		return true, nil
	}
	log.Warn().Msgf("Plugin %s not loaded for JVM %+v", plugin, jvm)
	return false, nil
}

func unloadAutoLoadPlugin(jvm *jvm.JavaVm, markerClass string, plugin string) {
	if HasClassLoaded(jvm, markerClass) {
		log.Trace().Msgf("Unloading plugin %s for JVM %+v", plugin, jvm)
		_, err := UnloadAgentPlugin(jvm, plugin)
		if err != nil {
			log.Warn().Msgf("Unloading plugin %s for JVM %+v failed: %s", plugin, jvm, err)
		}
	}
}

func UnloadAgentPlugin(jvm *jvm.JavaVm, plugin string) (bool, error) {
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
		plugin_tracking.Remove(jvm.Pid, plugin)
	}
	return unloaded, nil
}
func HasAgentPlugin(jvm *jvm.JavaVm, plugin string) bool {
	return plugin_tracking.Has(jvm.Pid, plugin)
}

func HasClassLoaded(jvm *jvm.JavaVm, className string) bool {
	return SendCommandToAgent(jvm, "class-loaded", className)
}

func setLogLevel(jvm *jvm.JavaVm, loglevel string) bool {
	return SendCommandToAgent(jvm, "log-level", loglevel)
}

func SendCommandToAgent(jvm *jvm.JavaVm, command string, args string) bool {
	return sendCommandToAgent(jvm, command, args, SocketTimeout)
}

func sendCommandToAgent(jvm *jvm.JavaVm, command string, args string, timeout time.Duration) bool {
	log.Info().Msgf("Sending command %s:%s to agent on PID %d", command, args, jvm.Pid)
	success := sendCommandToAgentViaSocket(jvm, command, args, timeout, func(rc string, response io.Reader) bool {
    resultMessage, err := GetCleanSocketCommandResult(response)
		log.Info().Msgf("Result from command %s:%s agent on PID %d: %s", command, args, jvm.Pid, resultMessage)
		if err != nil {
			log.Error().Msgf("Error reading result from command %s:%s agent on PID %d: %s", command, args, jvm.Pid, err)
			return false
		}
		if resultMessage != "" && rc == "OK" {
			log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned %s", command, args, jvm.Pid, resultMessage)
			return extutil.ToBool(resultMessage)
		} else {
			log.Warn().Msgf("Command '%s:%s' to agent on PID %d returned error: %s", command, args, jvm.Pid, resultMessage)
			return false
		}
	})
	return success != nil && *success
}

func SendCommandToAgentViaSocket[T any](jvm *jvm.JavaVm, command string, args string, handler func(rc string, response io.Reader) T) *T {
	return sendCommandToAgentViaSocket(jvm, command, args, SocketTimeout, handler)
}

func sendCommandToAgentViaSocket[T any](jvm *jvm.JavaVm, command string, args string, timeout time.Duration, handler func(rc string, response io.Reader) T) *T {
	pid := jvm.Pid
	connection := remote_jvm_connections.GetConnection(pid)
	if connection == nil {
		log.Debug().Msgf("RemoteJvmConnection from PID %d not found. Command '%s:%s' not sent.", pid, command, args)
		return nil
	}
  connection.Mutex.Lock()
  defer connection.Mutex.Unlock()

	d := net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", connection.Address())
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Error().Msgf("Error closing socket connection to JVM %d: %s", pid, err)
		}
	}(conn)
	if err != nil {
		if java_process.IsRunningProcess(pid) {
			log.Error().Msgf("Command '%s' could not be sent over socket to %+v (%s): %s", command, jvm, connection.Address(), err)
		} else {
			log.Debug().Msgf("Process %d not running anymore. Removing connection to %+v:%s", pid, jvm, connection.Address())
			remote_jvm_connections.RemoveConnection(pid)
		}

		return nil
	}
	err = conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		log.Error().Msgf("Error setting deadline for connection to JVM %d: %s", pid, err)
		return nil
	}
	err = conn.SetWriteDeadline(time.Now().Add(timeout))
	if err != nil {
		log.Error().Msgf("Error setting write deadline for connection to JVM %d: %s", pid, err)
		return nil
	}
	err = conn.SetReadDeadline(time.Now().Add(timeout))
	if err != nil {
		log.Error().Msgf("Error setting read deadline for connection to JVM %d: %s", pid, err)
		return nil
	}
	log.Trace().Msgf("Sending command '%s:%s' to agent on PID %d", command, args, pid)
	//_, err = conn.Write([]byte(command + ":" + args + "\n"))
	// Commands must end with newline
	_, err = fmt.Fprintf(conn, "%s:%s\n", command, args)
	if err != nil {
		log.Error().Msgf("Error sending command '%s:%s' to JVM %d: %s", command, args, pid, err)
		return nil
	}
	// First byte is always the return code
	rcByte := make([]byte, 1)
	_, err = conn.Read(rcByte)
	if err != nil {
		log.Error().Msgf("Error reading response return code from JVM %d: %s", pid, err)
		return nil
	}
	var rc string
	if rcByte[0] == 0 {
		rc = "OK"
	} else if rcByte[0] == 1 {
		rc = "ERROR"
	} else {
		rc = "UNKNOWN"
	}
	log.Info().Msgf("Return code from JVM %s for command %s:%s on pid %d", rc, command, args, pid)

	//scanner := bufio.NewScanner(conn)

	//var buf bytes.Buffer
	//_, err = io.Copy(&buf, conn)
	//if err != nil {
	//  log.Error().Msgf("Error reading response from JVM %d: %s", pid, err)
	//  return nil
	//}
	//trimmedBytes := bytes.Trim(resultBytes, "\000")
	//output, err := io.ReadAll(utfbom.SkipOnly(bytes.NewReader(trimmedBytes)))
	//if err != nil {
	//	log.Error().Msgf("Error reading response from JVM %d: %s", pid, err)
	//	return nil
	//}
	//message := string(output)
	//message = strings.Trim(message, "\n")
	return extutil.Ptr(handler(rc, conn))
}

func GetCleanSocketCommandResult(response io.Reader) (string, error) {
	resultMessage, err := bufio.NewReader(response).ReadString('\n')
	if err != nil {
		return "", err
	}
	output, err := io.ReadAll(utfbom.SkipOnly(bytes.NewReader([]byte(resultMessage))))
	if err != nil {
		return "", err
	}
	message := string(output)
	message = strings.Trim(message, "\n")
	return message, nil
}

func getJvmExtensionLogLevel() string {
	loglevel := os.Getenv("STEADYBIT_EXTENSION_LOG_LEVEL")
	if loglevel == "" {
		loglevel = "info"
	}
	return strings.ToUpper(loglevel)
}

func attachInternal(jvm *jvm.JavaVm) (bool, error) {
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

func isAttached(jvm *jvm.JavaVm) bool {
	connection := remote_jvm_connections.GetConnection(jvm.Pid)
	return connection != nil
}

func (j JavaExtensionFacade) AddedJvm(jvm *jvm.JavaVm) {
	attach(jvm, 5)
}

func attach(jvm *jvm.JavaVm, retries int) {
	attachJobs <- AttachJvmWork{jvm: jvm, retries: retries}
}

func (j JavaExtensionFacade) RemovedJvm(jvm *jvm.JavaVm) {
	//TODO: implement
	//abortAttach(jvm.Pid)
	plugin_tracking.RemoveAll(jvm.Pid)
	for _, listener := range attachedListeners {
		listener.AttachedProcessStopped(jvm)
	}
}

func AddAutoloadAgentPlugin(plugin string, markerClass string) {
	autoloadPluginsMutex.Lock()
	autoloadPlugins = append(autoloadPlugins, AutoloadPlugin{Plugin: plugin, MarkerClass: markerClass})
	autoloadPluginsMutex.Unlock()
	jvms.Range(func(key, value interface{}) bool {
		jvm := value.(*jvm.JavaVm)
		loadAutoLoadPlugin(jvm, markerClass, plugin)
		return true
	})
}

func loadAutoLoadPlugin(jvm *jvm.JavaVm, markerClass string, plugin string) {
	log.Info().Msgf("Autoloading plugin %s for %s", plugin, jvm.ToDebugString())
	if HasClassLoaded(jvm, markerClass) {
		log.Info().Msgf("Sending plugin %s for %s: %s", plugin, jvm.ToDebugString(), markerClass)
		scheduleLoadAgentPlugin(jvm, plugin, "", 5)
	}
}

func RemoveAutoloadAgentPlugin(plugin string, markerClass string) {
	for i, p := range autoloadPlugins {
		if p.Plugin == plugin && p.MarkerClass == markerClass {
			autoloadPluginsMutex.Lock()
			autoloadPlugins = append(autoloadPlugins[:i], autoloadPlugins[i+1:]...)
			autoloadPluginsMutex.Unlock()
			break
		}
	}
	jvms.Range(func(key, value interface{}) bool {
		jvm := value.(*jvm.JavaVm)
		unloadAutoLoadPlugin(jvm, plugin, markerClass)
		return true
	})
}
