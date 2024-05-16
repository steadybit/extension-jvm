package jvm

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/dimchansky/utfbom"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/config"
	"github.com/steadybit/extension-jvm/extjvm/java_process"
	"github.com/steadybit/extension-jvm/extjvm/plugin_tracking"
	"github.com/steadybit/extension-jvm/extjvm/remote_jvm_connections"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"github.com/steadybit/extension-kit/extutil"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type javaExtensionFacade struct{}

type attachJvmWork struct {
	jvm     *JavaVm
	retries int
}

type loadPluginJvmWork struct {
	jvm     *JavaVm
	plugin  string
	args    string
	retries int
}

type autoloadPlugin struct {
	MarkerClass string
	Plugin      string
}

type attachedListener interface {
	JvmAttachedSuccessfully(jvm *JavaVm)
	AttachedProcessStopped(jvm *JavaVm)
}

const (
	socketTimeout = 10 * time.Second
)

var (
	attachJobs        = make(chan attachJvmWork)
	attachedListeners []attachedListener
	loadPluginJobs    = make(chan loadPluginJvmWork)

	autoloadPluginsMutex sync.Mutex
	autoloadPlugins      = make([]autoloadPlugin, 0)

	javaagentInitJar = utils.GetJarPath("javaagent-init.jar")
	javaagentMainJar = utils.GetJarPath("javaagent-main.jar")
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
	addListener(&javaExtensionFacade{})
}

func AddAttachedListener(attachedListener attachedListener) {
	attachedListeners = append(attachedListeners, attachedListener)
	for _, discoveredJvm := range GetJvms() {
		attachedListener.JvmAttachedSuccessfully(&discoveredJvm)
	}
}

func attachWorker(attachJobs chan attachJvmWork) {
	for job := range attachJobs {
		job.retries--
		if job.retries > 0 {
			doAttach(job)
		} else {
			log.Warn().Msgf("attach retries for %s exceeded.", job.jvm.ToDebugString())
		}
	}
}

func loadPluginWorker(loadPluginJobs chan loadPluginJvmWork) {
	for job := range loadPluginJobs {
		job.retries--
		if job.retries > 0 {
			loadAgentPluginJob(job)
		} else {
			log.Warn().Msgf("Load Plugin retries for %s with plugin %s exceeded.", job.jvm.MainClass, job.plugin)
		}
	}
}

func doAttach(job attachJvmWork) {
	if err := attachInternal(job.jvm); err == nil {
		log.Info().Msgf("Successful attachment to JVM  %+v", job.jvm)

		loglevel := getJvmExtensionLogLevel()
		log.Trace().Msgf("Propagating Loglevel %s to Javaagent in JVM %+v", loglevel, job.jvm)
		if setLogLevel(job.jvm, loglevel) {
			log.Info().Msgf("Successfully set loglevel %s for JVM %+v", loglevel, job.jvm)
		} else {
			log.Debug().Msgf("Error setting loglevel %s for JVM %+v", loglevel, job.jvm)
			attach(job.jvm, job.retries)
		}

		for _, plugin := range autoloadPlugins {
			loadAutoLoadPlugin(job.jvm, plugin.MarkerClass, plugin.Plugin)
		}

		informListeners(job.jvm)
	} else if !java_process.IsRunningProcess(job.jvm.Pid) {
		log.Trace().Msgf("jvm stopped, attach failed. JVM %+v: %s", job.jvm, err)
	} else {
		log.Warn().Msgf("Error attaching to JVM %+v: %s", job.jvm, err)
		go func() {
			timeToSleep := 1
			if job.retries > 0 {
				timeToSleep = 60 / job.retries
			}
			time.Sleep(time.Duration(timeToSleep) * time.Second)
			// do retry
			attach(job.jvm, job.retries)
		}()
	}
}

func informListeners(vm *JavaVm) {
	for _, listener := range attachedListeners {
		listener := listener
		go func() {
			listener.JvmAttachedSuccessfully(vm)
		}()
	}
}

func scheduleLoadAgentPlugin(jvm *JavaVm, plugin string, args string, retries int) {
	loadPluginJobs <- loadPluginJvmWork{
		jvm:     jvm,
		plugin:  plugin,
		args:    args,
		retries: retries,
	}
}

func loadAgentPluginJob(job loadPluginJvmWork) {
	if err := LoadAgentPlugin(job.jvm, job.plugin, job.args); err != nil {
		log.Error().Msgf("Error loading plugin %s for JVM %+v: %s", job.plugin, job.jvm, err)
		go func() {
			time.Sleep(time.Duration(120/job.retries) * time.Second)
			// do retry
			scheduleLoadAgentPlugin(job.jvm, job.plugin, job.args, job.retries)
		}()
	}
}

func LoadAgentPlugin(jvm *JavaVm, plugin string, args string) error {
	if HasAgentPlugin(jvm, plugin) {
		log.Trace().Msgf("Plugin %s already loaded for JVM %+v", plugin, jvm)
		return nil
	}

	if _, err := os.Stat(plugin); err != nil {
		log.Error().Msgf("Plugin %s not found: %s", plugin, err)
		return err
	}

	var pluginPath string
	if jvm.IsRunningInContainer() {
		file := fmt.Sprintf("steadybit-%s", filepath.Base(plugin))
		files, err := GetAttachment(jvm).copyFiles("/tmp", map[string]string{file: plugin})
		if err != nil {
			log.Error().Msgf("Error copying plugin %s to container: %s", plugin, err)
			return err
		}
		pluginPath = files[file]
	} else {
		pluginPath = plugin
	}

	if success, err := SendCommandToAgentWithTimeout(jvm, "load-agent-plugin", fmt.Sprintf("%s,%s", pluginPath, args), time.Duration(30)*time.Second); success {
		log.Debug().Msgf("Plugin %s loaded for JVM %s", plugin, jvm.ToInfoString())
		plugin_tracking.Add(jvm.Pid, plugin)
		return nil
	} else {
		return err
	}
}

func unloadAutoLoadPlugin(jvm *JavaVm, markerClass string, plugin string) {
	if HasClassLoaded(jvm, markerClass) {
		log.Debug().Msgf("Unloading plugin %s for JVM %+v", plugin, jvm)

		if err := UnloadAgentPlugin(jvm, plugin); err != nil {
			log.Warn().Msgf("Unloading plugin %s for JVM %+v failed: %s", plugin, jvm, err)
		}
	}
}

func UnloadAgentPlugin(jvm *JavaVm, plugin string) error {
	_, err := os.Stat(plugin)
	if err != nil {
		log.Error().Msgf("Plugin %s not found: %s", plugin, err)
		return err
	}

	var args string
	if jvm.IsRunningInContainer() {
		file := fmt.Sprintf("steadybit-%s", filepath.Base(plugin))
		args = fmt.Sprintf("/tmp/%s,deleteFile=true", file)
	} else {
		args = plugin
	}

	if ok, err := SendCommandToAgent(jvm, "unload-agent-plugin", args); ok {
		plugin_tracking.Remove(jvm.Pid, plugin)
		return nil
	} else {
		return err
	}
}

func HasAgentPlugin(jvm *JavaVm, plugin string) bool {
	return plugin_tracking.Has(jvm.Pid, plugin)
}

func HasClassLoaded(jvm *JavaVm, className string) bool {
	result, err := SendCommandToAgent(jvm, "class-loaded", className)
	if err != nil {
		log.Error().Msgf("Error checking if class %s is loaded in JVM %s: %s", className, jvm.ToDebugString(), err)
		return false
	}
	return result
}

func setLogLevel(jvm *JavaVm, loglevel string) bool {
	result, err := SendCommandToAgent(jvm, "log-level", loglevel)
	if err != nil {
		log.Error().Msgf("Error setting loglevel %s in JVM %s: %s", loglevel, jvm.ToDebugString(), err)
		return false
	}
	return result
}

func SendCommandToAgent(jvm *JavaVm, command string, args string) (bool, error) {
	return SendCommandToAgentWithTimeout(jvm, command, args, socketTimeout)
}

func SendCommandToAgentWithHandler[T any](jvm *JavaVm, command string, args string, handler func(response io.Reader) (T, error)) (T, error) {
	return sendCommand(jvm, command, args, socketTimeout, handler)
}

func SendCommandToAgentWithTimeout(jvm *JavaVm, command string, args string, timeout time.Duration) (bool, error) {
	success, err := sendCommand(jvm, command, args, timeout, func(response io.Reader) (bool, error) {
		resultMessage, err := GetCleanSocketCommandResult(response)
		log.Debug().Msgf("Result from command %s:%s agent on PID %d: %s", command, args, jvm.Pid, resultMessage)
		if err != nil {
			return false, fmt.Errorf("cannot read result for command %s:%s agent on PID %d: %w", command, args, jvm.Pid, err)
		}
		if resultMessage != "" {
			return extutil.ToBool(resultMessage), nil
		} else {
			return false, errors.New("empty result")
		}
	})
	return success, err
}

func sendCommand[T any](jvm *JavaVm, command string, args string, timeout time.Duration, handler func(response io.Reader) (T, error)) (T, error) {
	var nilT T
	pid := jvm.Pid
	connection := remote_jvm_connections.GetConnection(pid)
	if connection == nil {
		return nilT, errors.New("connection not found")
	}
	connection.Mutex.Lock()
	defer connection.Mutex.Unlock()

	d := net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", connection.Address())
	defer func(conn net.Conn) {
		if conn == nil {
			return
		}
		err := conn.Close()
		if err != nil {
			log.Warn().Msgf("Error closing socket connection to JVM %d: %s", pid, err)
		}
	}(conn)

	if err != nil {
		if java_process.IsRunningProcess(pid) {
			return nilT, err
		} else {
			remote_jvm_connections.RemoveConnection(pid)
			return nilT, fmt.Errorf("process %d is not running anymore, connection failed: %w", pid, err)
		}
	}

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		log.Warn().Msgf("Error setting deadline for connection to JVM %d: %s", pid, err)
	}

	log.Debug().Msgf("Sending command '%s:%s' to agent on PID %d", command, args, pid)

	// Commands must end with newline
	if _, err = fmt.Fprintf(conn, "%s:%s\n", command, args); err != nil {
		log.Error().Msgf("Error sending command '%s:%s' to JVM %d: %s", command, args, pid, err)
		return nilT, fmt.Errorf("error sending command '%s:%s': %w", command, args, err)
	}

	// First byte is always the return code
	rcByte := make([]byte, 1)
	if _, err := conn.Read(rcByte); err != nil {
		return nilT, fmt.Errorf("error reading response return code: %w", err)
	}
	if rcByte[0] == 0 {
		return handler(utfbom.SkipOnly(conn))
	} else {
		return nilT, fmt.Errorf("command '%s:%s' returned rc: %d", command, args, rcByte[0])
	}
}

func GetCleanSocketCommandResult(response io.Reader) (string, error) {
	resultMessage, err := bufio.NewReader(response).ReadString('\n')
	if err != nil {
		return "", err
	}
	message := strings.Trim(resultMessage, "\n")
	return message, nil
}

func getJvmExtensionLogLevel() string {
	loglevel := os.Getenv("STEADYBIT_EXTENSION_LOG_LEVEL")
	if loglevel == "" {
		loglevel = "info"
	}
	return strings.ToUpper(loglevel)
}

func attachInternal(jvm *JavaVm) error {
	if isAttached(jvm) {
		log.Trace().Msgf("RemoteJvmConnection to JVM already established. %+v", jvm)
		return nil
	}

	log.Debug().Msgf("RemoteJvmConnection to JVM not found. Attaching now. %+v", jvm)

	if _, err := os.Stat(javaagentMainJar); err != nil {
		log.Error().Msgf("javaagentMainJar not found: %s", javaagentMainJar)
		return err
	}

	if _, err := os.Stat(javaagentInitJar); err != nil {
		log.Error().Msgf("javaagentInitJar not found: %s", javaagentInitJar)
		return err
	}

	if ok := GetAttachment(jvm).attach(javaagentMainJar, javaagentInitJar, int(config.Config.JavaAgentAttachmentPort)); !ok {
		return errors.New("could not attach to JVM")
	}

	if ok := remote_jvm_connections.WaitForConnection(jvm.Pid, time.Duration(90)*time.Second); !ok {
		log.Error().Msgf("JVM with did not call back after 90 seconds.")
		return errors.New("could not attach to JVM: VM with did not call back after 90 seconds")
	}

	return nil
}

func isAttached(jvm *JavaVm) bool {
	return remote_jvm_connections.GetConnection(jvm.Pid) != nil
}

func (j javaExtensionFacade) addedJvm(jvm *JavaVm) {
	attach(jvm, 5)
}

func attach(jvm *JavaVm, retries int) {
	attachJobs <- attachJvmWork{jvm: jvm, retries: retries}
}

func (j javaExtensionFacade) removedJvm(jvm *JavaVm) {
	plugin_tracking.RemoveAll(jvm.Pid)
	for _, listener := range attachedListeners {
		listener.AttachedProcessStopped(jvm)
	}
}

func AddAutoloadAgentPlugin(plugin string, markerClass string) {
	autoloadPluginsMutex.Lock()
	autoloadPlugins = append(autoloadPlugins, autoloadPlugin{Plugin: plugin, MarkerClass: markerClass})
	autoloadPluginsMutex.Unlock()
	vms := GetJvms()
	for _, vm := range vms {
		loadAutoLoadPlugin(&vm, markerClass, plugin)
	}
}

func loadAutoLoadPlugin(jvm *JavaVm, markerClass string, plugin string) {
	log.Info().Msgf("Autoloading plugin %s for %s", plugin, jvm.ToDebugString())
	if HasClassLoaded(jvm, markerClass) {
		log.Info().Msgf("Sending plugin %s for %s: %s", plugin, jvm.ToDebugString(), markerClass)
		scheduleLoadAgentPlugin(jvm, plugin, "", 6)
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
	jvMs := GetJvms()
	for _, vm := range jvMs {
		unloadAutoLoadPlugin(&vm, plugin, markerClass)
	}
}
