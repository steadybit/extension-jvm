package jvm

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/dimchansky/utfbom"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/java_process"
	"github.com/steadybit/extension-jvm/extjvm/plugin_tracking"
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

type JavaFacade struct {
	connections          jvmConnections
	autoloadPluginsMutex sync.Mutex
	autoloadPlugins      []autoloadPlugin
	attachJobs           chan attachJob
	attachListeners      []attachListener
	loadPluginJobs       chan loadPluginJob
	http                 *javaagentHttpServer
}

type attachJob struct {
	jvm     *JavaVm
	retries int
}

type loadPluginJob struct {
	jvm     *JavaVm
	plugin  string
	args    string
	retries int
}

type autoloadPlugin struct {
	MarkerClass string
	Plugin      string
}

type attachListener interface {
	JvmAttachedSuccessfully(jvm *JavaVm)
	AttachedProcessStopped(jvm *JavaVm)
}

const (
	socketTimeout = 10 * time.Second
)

var (
	javaagentInitJar = utils.GetJarPath("javaagent-init.jar")
	javaagentMainJar = utils.GetJarPath("javaagent-main.jar")
)

func (f *JavaFacade) Start() {
	attachmentEnabled := os.Getenv("STEADYBIT_EXTENSION_JVM_ATTACHMENT_ENABLED")
	if attachmentEnabled != "" && strings.ToLower(attachmentEnabled) != "true" {
		return
	}

	f.http = &javaagentHttpServer{connections: &f.connections}
	f.http.listen()

	f.attachJobs = make(chan attachJob, 50)
	for w := 1; w <= 4; w++ {
		go f.attachWorker(f.attachJobs)
	}

	f.loadPluginJobs = make(chan loadPluginJob, 50)
	for w := 1; w <= 4; w++ {
		go f.loadPluginWorker(f.loadPluginJobs)
	}
	addListener(f)
}

func (f *JavaFacade) Stop() {
	removeListener(f)
	if f.http != nil {
		f.http.shutdown()
	}
}

func (f *JavaFacade) AddAttachedListener(attachedListener attachListener) {
	f.attachListeners = append(f.attachListeners, attachedListener)
	for _, discoveredJvm := range GetJvms() {
		attachedListener.JvmAttachedSuccessfully(&discoveredJvm)
	}
}

func (f *JavaFacade) attachWorker(attachJobs chan attachJob) {
	for job := range attachJobs {
		job.retries--
		if job.retries > 0 {
			f.doAttach(job)
		} else {
			log.Warn().Msgf("attach retries for %s exceeded.", job.jvm.ToDebugString())
		}
	}
}

func (f *JavaFacade) loadPluginWorker(loadPluginJobs chan loadPluginJob) {
	for job := range loadPluginJobs {
		job.retries--
		if job.retries > 0 {
			f.loadAgentPluginJob(job)
		} else {
			log.Warn().Msgf("Load Plugin retries for %s with plugin %s exceeded.", job.jvm.MainClass, job.plugin)
		}
	}
}

func (f *JavaFacade) doAttach(job attachJob) {
	if err := f.attachInternal(job.jvm); err == nil {
		log.Info().Msgf("Successful attachment to JVM  %+v", job.jvm)

		loglevel := getJvmExtensionLogLevel()
		log.Trace().Msgf("Propagating Loglevel %s to Javaagent in JVM %+v", loglevel, job.jvm)
		if f.setLogLevel(job.jvm, loglevel) {
			log.Info().Msgf("Successfully set loglevel %s for JVM %+v", loglevel, job.jvm)
		} else {
			log.Debug().Msgf("Error setting loglevel %s for JVM %+v", loglevel, job.jvm)
			f.attach(job.jvm, job.retries)
		}

		for _, plugin := range f.autoloadPlugins {
			f.loadAutoLoadPlugin(job.jvm, plugin.MarkerClass, plugin.Plugin)
		}

		f.informListeners(job.jvm)
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
			f.attach(job.jvm, job.retries)
		}()
	}
}

func (f *JavaFacade) informListeners(vm *JavaVm) {
	for _, listener := range f.attachListeners {
		listener := listener
		go func() {
			listener.JvmAttachedSuccessfully(vm)
		}()
	}
}

func (f *JavaFacade) scheduleLoadAgentPlugin(jvm *JavaVm, plugin string, args string, retries int) {
	f.loadPluginJobs <- loadPluginJob{
		jvm:     jvm,
		plugin:  plugin,
		args:    args,
		retries: retries,
	}
}

func (f *JavaFacade) loadAgentPluginJob(job loadPluginJob) {
	if err := f.LoadAgentPlugin(job.jvm, job.plugin, job.args); err != nil {
		log.Error().Msgf("Error loading plugin %s for JVM %+v: %s", job.plugin, job.jvm, err)
		go func() {
			time.Sleep(time.Duration(120/job.retries) * time.Second)
			// do retry
			f.scheduleLoadAgentPlugin(job.jvm, job.plugin, job.args, job.retries)
		}()
	}
}

func (f *JavaFacade) LoadAgentPlugin(jvm *JavaVm, plugin string, args string) error {
	if f.HasAgentPlugin(jvm, plugin) {
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

	if success, err := f.SendCommandToAgentWithTimeout(jvm, "load-agent-plugin", fmt.Sprintf("%s,%s", pluginPath, args), time.Duration(30)*time.Second); success {
		log.Debug().Msgf("Plugin %s loaded for JVM %s", plugin, jvm.ToInfoString())
		plugin_tracking.Add(jvm.Pid, plugin)
		return nil
	} else {
		return err
	}
}

func (f *JavaFacade) unloadAutoLoadPlugin(jvm *JavaVm, markerClass string, plugin string) {
	if f.HasClassLoaded(jvm, markerClass) {
		log.Debug().Msgf("Unloading plugin %s for JVM %+v", plugin, jvm)

		if err := f.UnloadAgentPlugin(jvm, plugin); err != nil {
			log.Warn().Msgf("Unloading plugin %s for JVM %+v failed: %s", plugin, jvm, err)
		}
	}
}

func (f *JavaFacade) UnloadAgentPlugin(jvm *JavaVm, plugin string) error {
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

	if ok, err := f.SendCommandToAgent(jvm, "unload-agent-plugin", args); ok {
		plugin_tracking.Remove(jvm.Pid, plugin)
		return nil
	} else {
		return err
	}
}

func (f *JavaFacade) HasAgentPlugin(jvm *JavaVm, plugin string) bool {
	return plugin_tracking.Has(jvm.Pid, plugin)
}

func (f *JavaFacade) HasClassLoaded(jvm *JavaVm, className string) bool {
	result, err := f.SendCommandToAgent(jvm, "class-loaded", className)
	if err != nil {
		log.Error().Msgf("Error checking if class %s is loaded in JVM %s: %s", className, jvm.ToDebugString(), err)
		return false
	}
	return result
}

func (f *JavaFacade) setLogLevel(jvm *JavaVm, loglevel string) bool {
	result, err := f.SendCommandToAgent(jvm, "log-level", loglevel)
	if err != nil {
		log.Error().Msgf("Error setting loglevel %s in JVM %s: %s", loglevel, jvm.ToDebugString(), err)
		return false
	}
	return result
}

func (f *JavaFacade) SendCommandToAgent(jvm *JavaVm, command string, args string) (bool, error) {
	return f.SendCommandToAgentWithTimeout(jvm, command, args, socketTimeout)
}

func (f *JavaFacade) SendCommandToAgentWithHandler(jvm *JavaVm, command string, args string, handler func(response io.Reader) (any, error)) (any, error) {
	return f.sendCommand(jvm, command, args, socketTimeout, handler)
}

func (f *JavaFacade) SendCommandToAgentWithTimeout(jvm *JavaVm, command string, args string, timeout time.Duration) (bool, error) {
	success, err := f.sendCommand(jvm, command, args, timeout, func(response io.Reader) (any, error) {
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
	if success != nil {
		return success.(bool), err
	}
	return false, err
}

func (f *JavaFacade) sendCommand(jvm *JavaVm, command string, args string, timeout time.Duration, handler func(response io.Reader) (any, error)) (any, error) {
	pid := jvm.Pid
	connection := f.connections.getConnection(pid)
	if connection == nil {
		return nil, errors.New("connection not found")
	}
	connection.Lock()
	defer connection.Unlock()

	d := net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", connection.Address)
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
			return nil, err
		} else {
			f.connections.removeConnection(pid)
			return nil, fmt.Errorf("process %d is not running anymore, connection failed: %w", pid, err)
		}
	}

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		log.Warn().Msgf("Error setting deadline for connection to JVM %d: %s", pid, err)
	}

	log.Debug().Msgf("Sending command '%s:%s' to agent on PID %d", command, args, pid)

	// Commands must end with newline
	if _, err = fmt.Fprintf(conn, "%s:%s\n", command, args); err != nil {
		log.Error().Msgf("Error sending command '%s:%s' to JVM %d: %s", command, args, pid, err)
		return nil, fmt.Errorf("error sending command '%s:%s': %w", command, args, err)
	}

	// First byte is always the return code
	rcByte := make([]byte, 1)
	if _, err := conn.Read(rcByte); err != nil {
		return nil, fmt.Errorf("error reading response return code: %w", err)
	}
	if rcByte[0] == 0 {
		return handler(utfbom.SkipOnly(conn))
	} else {
		return nil, fmt.Errorf("command '%s:%s' returned rc: %d", command, args, rcByte[0])
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

func (f *JavaFacade) attachInternal(jvm *JavaVm) error {
	if f.isAttached(jvm) {
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

	if ok := GetAttachment(jvm).attach(javaagentMainJar, javaagentInitJar, f.http.port); !ok {
		return errors.New("could not attach to JVM")
	}

	if ok := f.connections.waitForConnection(jvm.Pid, time.Duration(90)*time.Second); !ok {
		log.Error().Msgf("JVM with did not call back after 90 seconds.")
		return errors.New("could not attach to JVM: VM with did not call back after 90 seconds")
	}

	return nil
}

func (f *JavaFacade) isAttached(jvm *JavaVm) bool {
	return f.connections.getConnection(jvm.Pid) != nil
}

func (f *JavaFacade) addedJvm(jvm *JavaVm) {
	f.attach(jvm, 5)
}

func (f *JavaFacade) attach(jvm *JavaVm, retries int) {
	f.attachJobs <- attachJob{jvm: jvm, retries: retries}
}

func (f *JavaFacade) removedJvm(jvm *JavaVm) {
	plugin_tracking.RemoveAll(jvm.Pid)
	for _, listener := range f.attachListeners {
		listener.AttachedProcessStopped(jvm)
	}
}

func (f *JavaFacade) AddAutoloadAgentPlugin(plugin string, markerClass string) {
	f.autoloadPluginsMutex.Lock()
	f.autoloadPlugins = append(f.autoloadPlugins, autoloadPlugin{Plugin: plugin, MarkerClass: markerClass})
	f.autoloadPluginsMutex.Unlock()
	vms := GetJvms()
	for _, vm := range vms {
		f.loadAutoLoadPlugin(&vm, markerClass, plugin)
	}
}

func (f *JavaFacade) loadAutoLoadPlugin(jvm *JavaVm, markerClass string, plugin string) {
	log.Info().Msgf("Autoloading plugin %s for %s", plugin, jvm.ToDebugString())
	if f.HasClassLoaded(jvm, markerClass) {
		log.Info().Msgf("Sending plugin %s for %s: %s", plugin, jvm.ToDebugString(), markerClass)
		f.scheduleLoadAgentPlugin(jvm, plugin, "", 6)
	}
}

func (f *JavaFacade) RemoveAutoloadAgentPlugin(plugin string, markerClass string) {
	for i, p := range f.autoloadPlugins {
		if p.Plugin == plugin && p.MarkerClass == markerClass {
			f.autoloadPluginsMutex.Lock()
			f.autoloadPlugins = append(f.autoloadPlugins[:i], f.autoloadPlugins[i+1:]...)
			f.autoloadPluginsMutex.Unlock()
			break
		}
	}
	for _, vm := range GetJvms() {
		f.unloadAutoLoadPlugin(&vm, plugin, markerClass)
	}
}
