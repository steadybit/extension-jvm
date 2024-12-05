package jvm

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/dimchansky/utfbom"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/config"
	"github.com/steadybit/extension-jvm/extjvm/jvm/hsperf"
	"github.com/steadybit/extension-jvm/extjvm/jvm/internal"
	"github.com/steadybit/extension-jvm/extjvm/jvm/jvmprocess"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"github.com/steadybit/extension-kit/extutil"
	"io"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

type JavaFacade interface {
	Start()
	Stop()
	AddAttachedListener(AttachedListener AttachListener)
	RemoveAttachedListener(AttachedListener AttachListener)
	LoadAgentPlugin(javaVm JavaVm, plugin string, args string) error
	UnloadAgentPlugin(javaVm JavaVm, plugin string) error
	HasAgentPlugin(javaVm JavaVm, plugin string) bool
	HasClassLoaded(javaVm JavaVm, className string) bool
	SendCommandToAgent(javaVm JavaVm, command string, args string) (bool, error)
	SendCommandToAgentWithHandler(javaVm JavaVm, command string, args string, handler func(response io.Reader) (any, error)) (any, error)
	SendCommandToAgentWithTimeout(javaVm JavaVm, command string, args string, timeout time.Duration) (bool, error)
	AddAutoloadAgentPlugin(plugin string, markerClass string)
	RemoveAutoloadAgentPlugin(plugin string, markerClass string)
	GetJvm(pid int32) JavaVm
	GetJvms() []JavaVm
}

type defaultJavaFacade struct {
	javaVms                   *javaVms
	connections               jvmConnections
	autoloadPluginsMutex      sync.Mutex
	autoloadPlugins           []autoloadPlugin
	attachJobs                chan attachJob
	attachListeners           []AttachListener
	loadPluginJobs            chan loadPluginJob
	http                      *javaagentHttpServer
	processWatcher            jvmprocess.ProcessWatcher
	hsperfWatcher             hsperf.Watcher
	inspector                 JavaProcessInspector
	plugins                   internal.PluginMap
	minProcessAgeBeforeAttach time.Duration
}

type attachJob struct {
	jvm     JavaVm
	retries int
}

type loadPluginJob struct {
	jvm     JavaVm
	plugin  string
	args    string
	retries int
}

type autoloadPlugin struct {
	MarkerClass string
	Plugin      string
}

type AttachListener interface {
	Attached(javaVm JavaVm)
	Detached(javaVm JavaVm)
}

const (
	socketTimeout = 10 * time.Second
)

var (
	javaagentInitJar = utils.GetJarPath("javaagent-init.jar")
	javaagentMainJar = utils.GetJarPath("javaagent-main.jar")
)

func NewJavaFacade() JavaFacade {
	return &defaultJavaFacade{
		processWatcher: jvmprocess.ProcessWatcher{Interval: 5 * time.Second},
	}
}

func (f *defaultJavaFacade) Start() {
	if !config.Config.JvmAttachmentEnabled {
		log.Warn().Msg("JVM attachment is disabled.")
		return
	}

	f.minProcessAgeBeforeAttach = config.Config.MinProcessAgeBeforeAttach
	f.http = &javaagentHttpServer{connections: &f.connections}
	f.http.listen()
	f.javaVms = newJavaVms()

	f.attachJobs = make(chan attachJob, 50)
	for w := 1; w <= 4; w++ {
		go f.attachWorker(f.attachJobs)
	}

	f.loadPluginJobs = make(chan loadPluginJob, 50)
	for w := 1; w <= 4; w++ {
		go f.loadPluginWorker(f.loadPluginJobs)
	}

	f.inspector.minProcessAgeBeforeInspect = config.Config.MinProcessAgeBeforeInspect
	f.inspector.Start()
	f.hsperfWatcher.Start()
	f.processWatcher.Start()

	go func() {
		for {
			select {
			case p := <-f.hsperfWatcher.Processes:
				f.inspector.Inspect(p, 5, "hsperfdata")
			case p := <-f.processWatcher.Processes:
				f.inspector.Inspect(p, 5, "os-process")
			}
		}
	}()

	go func() {
		for javaVm := range f.inspector.JavaVms {
			f.javaVms.addJvm(javaVm)
		}
	}()

	go func() {
		for javaVm := range f.javaVms.Added {
			sleep := 0 * time.Millisecond
			age := time.Since(javaVm.CreateTime())
			if age < f.minProcessAgeBeforeAttach {
				sleep = f.minProcessAgeBeforeAttach - age
			}

			go func() {
				time.Sleep(sleep)
				f.attach(javaVm, 5)
			}()
		}
	}()

	go func() {
		for javaVm := range f.javaVms.Removed {
			f.plugins.RemoveAll(javaVm.Pid())
			for _, l := range f.attachListeners {
				l.Detached(javaVm)
			}
		}
	}()
}

func (f *defaultJavaFacade) Stop() {
	f.processWatcher.Stop()
	f.hsperfWatcher.Stop()
	f.inspector.Stop()
	if f.http != nil {
		f.http.shutdown()
	}
}

func (f *defaultJavaFacade) AddAttachedListener(attachedListener AttachListener) {
	f.attachListeners = append(f.attachListeners, attachedListener)
	for _, discoveredJvm := range f.javaVms.getJvms() {
		attachedListener.Attached(discoveredJvm)
	}
}

func (f *defaultJavaFacade) RemoveAttachedListener(attachedListener AttachListener) {
	f.attachListeners = slices.DeleteFunc(f.attachListeners, func(l AttachListener) bool {
		return l == attachedListener
	})
}

func (f *defaultJavaFacade) attachWorker(attachJobs chan attachJob) {
	for job := range attachJobs {
		job.retries--
		if job.retries > 0 {
			f.doAttach(job)
		} else {
			log.Warn().Msgf("attach retries for %s exceeded.", job.jvm.ToDebugString())
		}
	}
}

func (f *defaultJavaFacade) loadPluginWorker(loadPluginJobs chan loadPluginJob) {
	for job := range loadPluginJobs {
		job.retries--
		if job.retries > 0 {
			f.loadAgentPluginJob(job)
		} else {
			log.Warn().Msgf("Load Plugin retries for %s with plugin %s exceeded.", job.jvm.MainClass(), job.plugin)
		}
	}
}

func (f *defaultJavaFacade) doAttach(job attachJob) {
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

		for _, l := range f.attachListeners {
			l.Attached(job.jvm)
		}
	} else if !job.jvm.IsRunning() {
		log.Trace().Msgf("jvm stopped, attach failed. JVM %s", job.jvm.ToInfoString())
	} else {
		log.Warn().Err(err).Msgf("Error attaching to JVM %s", job.jvm.ToInfoString())
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

func (f *defaultJavaFacade) scheduleLoadAgentPlugin(javaVm JavaVm, plugin string, args string, retries int) {
	f.loadPluginJobs <- loadPluginJob{
		jvm:     javaVm,
		plugin:  plugin,
		args:    args,
		retries: retries,
	}
}

func (f *defaultJavaFacade) loadAgentPluginJob(job loadPluginJob) {
	if err := f.LoadAgentPlugin(job.jvm, job.plugin, job.args); err != nil {
		log.Error().Msgf("Error loading plugin %s for JVM %+v: %s", job.plugin, job.jvm, err)
		go func() {
			time.Sleep(time.Duration(120/job.retries) * time.Second)
			// do retry
			f.scheduleLoadAgentPlugin(job.jvm, job.plugin, job.args, job.retries)
		}()
	}
}

func (f *defaultJavaFacade) LoadAgentPlugin(javaVm JavaVm, plugin string, args string) error {
	if f.HasAgentPlugin(javaVm, plugin) {
		log.Trace().Msgf("Plugin %s already loaded for JVM %+v", plugin, javaVm)
		return nil
	}

	if _, err := os.Stat(plugin); err != nil {
		log.Error().Msgf("Plugin %s not found: %s", plugin, err)
		return err
	}

	var pluginPath string
	var attachment = GetAttachment(javaVm)
	if !attachment.canAccessHostFiles() {
		file := fmt.Sprintf("steadybit-%s", filepath.Base(plugin))
		files, err := attachment.copyFiles("/tmp", map[string]string{file: plugin})
		if err != nil {
			log.Error().Msgf("Error copying plugin %s to container: %s", plugin, err)
			return err
		}
		pluginPath = files[file]
	} else {
		pluginPath = plugin
	}

	if success, err := f.SendCommandToAgentWithTimeout(javaVm, "load-agent-plugin", fmt.Sprintf("%s,%s", pluginPath, args), time.Duration(30)*time.Second); success {
		log.Debug().Msgf("Plugin %s loaded for JVM %s", plugin, javaVm.ToInfoString())
		f.plugins.Add(javaVm.Pid(), plugin)
		return nil
	} else {
		return err
	}
}

func (f *defaultJavaFacade) unloadAutoLoadPlugin(javaVm JavaVm, markerClass string, plugin string) {
	if f.HasClassLoaded(javaVm, markerClass) {
		log.Debug().Msgf("Unloading plugin %s for JVM %+v", plugin, javaVm)

		if err := f.UnloadAgentPlugin(javaVm, plugin); err != nil {
			log.Warn().Msgf("Unloading plugin %s for JVM %+v failed: %s", plugin, javaVm, err)
		}
	}
}

func (f *defaultJavaFacade) UnloadAgentPlugin(javaVm JavaVm, plugin string) error {
	_, err := os.Stat(plugin)
	if err != nil {
		log.Error().Msgf("Plugin %s not found: %s", plugin, err)
		return err
	}

	var args string
	if !GetAttachment(javaVm).canAccessHostFiles() {
		args = fmt.Sprintf("/tmp/steadybit-%s,deleteFile=true", filepath.Base(plugin))
	} else {
		args = plugin
	}

	if ok, err := f.SendCommandToAgent(javaVm, "unload-agent-plugin", args); ok {
		f.plugins.Remove(javaVm.Pid(), plugin)
		return nil
	} else {
		return err
	}
}

func (f *defaultJavaFacade) HasAgentPlugin(javaVm JavaVm, plugin string) bool {
	return f.plugins.Has(javaVm.Pid(), plugin)
}

func (f *defaultJavaFacade) HasClassLoaded(javaVm JavaVm, className string) bool {
	result, err := f.SendCommandToAgent(javaVm, "class-loaded", className)
	if err != nil {
		log.Error().Msgf("Error checking if class %s is loaded in JVM %s: %s", className, javaVm.ToDebugString(), err)
		return false
	}
	return result
}

func (f *defaultJavaFacade) setLogLevel(javaVm JavaVm, loglevel string) bool {
	result, err := f.SendCommandToAgent(javaVm, "log-level", loglevel)
	if err != nil {
		log.Error().Msgf("Error setting loglevel %s in JVM %s: %s", loglevel, javaVm.ToDebugString(), err)
		return false
	}
	return result
}

func (f *defaultJavaFacade) SendCommandToAgent(javaVm JavaVm, command string, args string) (bool, error) {
	return f.SendCommandToAgentWithTimeout(javaVm, command, args, socketTimeout)
}

func (f *defaultJavaFacade) SendCommandToAgentWithHandler(javaVm JavaVm, command string, args string, handler func(response io.Reader) (any, error)) (any, error) {
	return f.sendCommand(javaVm, command, args, socketTimeout, handler)
}

func (f *defaultJavaFacade) SendCommandToAgentWithTimeout(javaVm JavaVm, command string, args string, timeout time.Duration) (bool, error) {
	success, err := f.sendCommand(javaVm, command, args, timeout, func(response io.Reader) (any, error) {
		resultMessage, err := GetCleanSocketCommandResult(response)
		log.Debug().Msgf("Result from command %s:%s agent on PID %d: %s", command, args, javaVm.Pid(), resultMessage)
		if err != nil {
			return false, fmt.Errorf("cannot read result for command %s:%s agent on PID %d: %w", command, args, javaVm.Pid(), err)
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

func (f *defaultJavaFacade) sendCommand(javaVm JavaVm, command string, args string, timeout time.Duration, handler func(response io.Reader) (any, error)) (any, error) {
	connection := f.connections.getConnection(javaVm.Pid())
	if connection == nil {
		return nil, errors.New("connection not found")
	}
	connection.lock()
	defer connection.unlock()

	d := net.Dialer{Timeout: timeout}
	conn, err := d.Dial("tcp", connection.Address)
	defer func(conn net.Conn) {
		if conn == nil {
			return
		}
		err := conn.Close()
		if err != nil {
			log.Warn().Msgf("Error closing socket connection to JVM %d: %s", javaVm.Pid(), err)
		}
	}(conn)

	if err != nil {
		if javaVm.IsRunning() {
			return nil, err
		} else {
			f.connections.removeConnection(javaVm.Pid())
			return nil, fmt.Errorf("process %d is not running anymore, connection failed: %w", javaVm.Pid(), err)
		}
	}

	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		log.Warn().Msgf("Error setting deadline for connection to JVM %d: %s", javaVm.Pid(), err)
	}

	log.Trace().Msgf("Sending command '%s:%s' to agent on PID %d", command, args, javaVm.Pid())

	// Commands must end with newline
	if _, err = fmt.Fprintf(conn, "%s:%s\n", command, args); err != nil {
		log.Error().Msgf("Error sending command '%s:%s' to JVM %d: %s", command, args, javaVm.Pid(), err)
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
	loglevel := config.Config.JavaAgentLogLevel
	if loglevel == "" {
		loglevel = "info"
	}
	return strings.ToUpper(loglevel)
}

func (f *defaultJavaFacade) attachInternal(javaVm JavaVm) error {
	if f.isAttached(javaVm) {
		log.Trace().Msgf("RemoteJvmConnection to JVM already established. %s", javaVm.ToInfoString())
		return nil
	}

	log.Debug().Msgf("RemoteJvmConnection to JVM not found. Attaching now. %s", javaVm.ToInfoString())

	if _, err := os.Stat(javaagentMainJar); err != nil {
		log.Error().Msgf("javaagentMainJar not found: %s", javaagentMainJar)
		return err
	}

	if _, err := os.Stat(javaagentInitJar); err != nil {
		log.Error().Msgf("javaagentInitJar not found: %s", javaagentInitJar)
		return err
	}

	if ok := GetAttachment(javaVm).attach(javaagentMainJar, javaagentInitJar, f.http.port); !ok {
		return errors.New("could not attach to JVM")
	}

	if ok := f.connections.waitForConnection(javaVm.Pid(), time.Duration(90)*time.Second); !ok {
		log.Error().Msgf("JVM with did not call back after 90 seconds.")
		return errors.New("could not attach to JVM: VM with did not call back after 90 seconds")
	}

	return nil
}

func (f *defaultJavaFacade) isAttached(jvm JavaVm) bool {
	return f.connections.getConnection(jvm.Pid()) != nil
}

func (f *defaultJavaFacade) attach(jvm JavaVm, retries int) {
	f.attachJobs <- attachJob{jvm: jvm, retries: retries}
}

func (f *defaultJavaFacade) AddAutoloadAgentPlugin(plugin string, markerClass string) {
	f.autoloadPluginsMutex.Lock()
	f.autoloadPlugins = append(f.autoloadPlugins, autoloadPlugin{Plugin: plugin, MarkerClass: markerClass})
	f.autoloadPluginsMutex.Unlock()
	for _, jvm := range f.javaVms.getJvms() {
		f.loadAutoLoadPlugin(jvm, markerClass, plugin)
	}
}

func (f *defaultJavaFacade) loadAutoLoadPlugin(javaVm JavaVm, markerClass string, plugin string) {
	if f.HasClassLoaded(javaVm, markerClass) {
		log.Info().Msgf("Autoloading plugin %s with marker class %s for %s", plugin, markerClass, javaVm.ToInfoString())
		f.scheduleLoadAgentPlugin(javaVm, plugin, "", 6)
	}
}

func (f *defaultJavaFacade) RemoveAutoloadAgentPlugin(plugin string, markerClass string) {
	for i, p := range f.autoloadPlugins {
		if p.Plugin == plugin && p.MarkerClass == markerClass {
			f.autoloadPluginsMutex.Lock()
			f.autoloadPlugins = append(f.autoloadPlugins[:i], f.autoloadPlugins[i+1:]...)
			f.autoloadPluginsMutex.Unlock()
			break
		}
	}
	for _, jvm := range f.javaVms.getJvms() {
		f.unloadAutoLoadPlugin(jvm, plugin, markerClass)
	}
}

func (f *defaultJavaFacade) GetJvm(pid int32) JavaVm {
	return f.javaVms.getJvm(pid)
}

func (f *defaultJavaFacade) GetJvms() []JavaVm {
	return f.javaVms.getJvms()
}
