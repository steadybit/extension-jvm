package extjvm

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/process"
	"github.com/steadybit/extension-jvm/extjvm/hotspot"
	"github.com/steadybit/extension-jvm/extjvm/hsperf"
	"github.com/steadybit/extension-jvm/extjvm/java_process"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-jvm/extjvm/procfs"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"github.com/steadybit/extension-kit/extruntime"
	"github.com/xin053/hsperfdata"
	"golang.org/x/sys/unix"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

var (
	jvms                sync.Map //map[int32]java_process.JavaVm
	ClasspathExcludes   = []string{"JetBrains Toolbox.app", "IntelliJ IDEA", "surefirebooter", "Eclipse"}
	CommandlineExcludes = []string{"IntelliJ IDEA", "com.intellij.idea.Main", "jetbrains.buildServer.agent.Launcher",
		"jetbrains.buildServer.agent.AgentMain", "org.jetbrains.jps.cmdline.BuildMain", "org.jetbrains.idea.maven.server.RemoteMavenServer",
		"org.jetbrains.jps.cmdline.Launcher", "org.jetbrains.plugins.scala.nailgun.NailgunRunner", "sun.tools.",
		"com.steadybit.javaagent.ExternalJavaagentAttachment", "steadybit.agent.disable-jvm-attachment",
		"-XX:+DisableAttachMechanism", "-Dcom.ibm.tools.attach.enable=no", "com.steadybit.SteadybitAgentApplication"}
	listeners []Listener
)

type Listener interface {
	AddedJvm(jvm *jvm.JavaVm)
	RemovedJvm(jvm *jvm.JavaVm)
}

type JavaVMS struct{}

func AddListener(listener Listener) {
	listeners = append(listeners, listener)
	for _, vm := range GetJVMs() {
		listener.AddedJvm(&vm)
	}
}

func RemoveListener(listener Listener) {
	for i, l := range listeners {
		if l == listener {
			listeners = append(listeners[:i], listeners[i+1:]...)
			break
		}
	}
}

func addJVMListener() {
	java_process.AddListener(&JavaVMS{})
	hotspot.AddListener(&JavaVMS{})
}

func (j *JavaVMS) NewJavaProcess(p *process.Process) bool {
	_, ok := jvms.Load(p.Pid)
	if !ok {
		if java_process.IsRunning(p) {
			vm := createJvm(p)
			if vm != nil {
				log.Info().Msgf("Discovered JVM %s via process with pid %d", vm.VmName, p.Pid)
				addJvm(vm)
				return true
			}
			return false
		} else {
			log.Trace().Msgf("Process %d is not running anymore", p.Pid)
			return false
		}
	}
	return true
}

func (j *JavaVMS) NewHotspotProcess(p *process.Process) bool {
	_, ok := jvms.Load(p.Pid)
	if !ok && java_process.IsRunning(p) {
		vm := createJvm(p)
		if vm != nil {
			log.Info().Msgf("Discovered JVM %s via hotspot with pid %d", vm.VmName, p.Pid)
			addJvm(vm)
			return true
		}
		return false
	}
	return true
}

func addJvm(jvm *jvm.JavaVm) {
	if jvm == nil || isExcluded(jvm) {
		return
	}
	log.Debug().Msgf("Discovered JVM %s", jvm.ToDebugString())
	jvms.Store(jvm.Pid, *jvm)
	for _, listener := range listeners {
		listener.AddedJvm(jvm)
	}
}

func GetJVMs() []jvm.JavaVm {
	removeStoppedJvms()
	result := make([]jvm.JavaVm, 0)
	jvms.Range(func(key, value interface{}) bool {
		result = append(result, value.(jvm.JavaVm))
		return true
	})
	return result
}

func removeStoppedJvms() {
	jvms.Range(func(key, value interface{}) bool {
		vm := value.(jvm.JavaVm)
		p, err := process.NewProcess(vm.Pid)
		if err != nil {
			log.Trace().Err(err).Msg("Process not found: " + strconv.Itoa(int(vm.Pid)) + " - removing from JVMs. Error: " + err.Error())
			removeJVM(key, vm)
			return true
		}
		if !java_process.IsRunning(p) {
			log.Trace().Msgf("Process not running: %s: %s - removing from JVMs ", strconv.Itoa(int(vm.Pid)), vm.MainClass)
			removeJVM(key, vm)
		}
		return true
	})
}

func removeJVM(key interface{}, vm jvm.JavaVm) {
	jvms.Delete(key)
	java_process.RemovePidFromDiscoveredPids(vm.Pid)
	log.Debug().Msgf("Removing JVM %s", vm.ToDebugString())
	for _, listener := range listeners {
		listener.RemovedJvm(&vm)
	}
}
func createJvm(p *process.Process) *jvm.JavaVm {
	containerId := procfs.GetContainerIdForProcess(p)
	if containerId == "" {
		return createHostJvm(p)
	}

	containerPid := procfs.GetContainerPid(p.Pid)
	if containerPid > 0 {
		return createContainerizedJvm(p, containerId, containerPid, procfs.GetProcessRoot(p.Pid))
	}
	return nil
}

func createContainerizedJvm(p *process.Process, containerId string, containerPid int32, containerFs string) *jvm.JavaVm {
	log.Debug().Msgf("Found containerized JVM %s with containerPid %d on FS %s", containerId, containerPid, containerFs)
	filePaths := hotspot.GetRootHsPerfPaths(p.Pid, containerFs)
	if len(filePaths) == 0 {
		log.Warn().Msgf("Could not find hsperfdata root path for container %s on pid %d. Will be retried", containerId, p.Pid)
		return nil
	}
	hsPerfDataPath := filePaths[strconv.Itoa(int(containerPid))]
	if hsPerfDataPath == "" {
		log.Debug().Msgf("Could not find hsperfdata path for container %s", containerId)
		return nil
	}
	javaVm := parsePerfDataBuffer(p, hsPerfDataPath)
	if javaVm == nil {
		log.Warn().Msgf("Could not parse hsperfdata for container %s", containerId)
		return nil
	}
	javaVm.InContainerPid = int(containerPid)
	javaVm.ContainerId = containerId
	return javaVm
}

func createHostJvm(p *process.Process) *jvm.JavaVm {
	if runtime.GOOS != "windows" {
		rootPath := procfs.GetProcessRoot(p.Pid)
		dirsGlob := filepath.Join(rootPath, os.TempDir())
		vm := findJvmOnPath(p, dirsGlob)
		if vm != nil {
			return vm
		}
	}

	//find via jvm hsperfdata using an alternative tempdir
	cmdline, err := p.Cmdline()
	if err == nil && cmdline != "" {
		arg := strings.Split(cmdline, " ")
		if strings.HasPrefix(arg[0], "-Djava.io.tmpdir") {
			tokens := strings.Split(arg[0], "=")
			if len(tokens) > 1 {
				dirsGlob := filepath.Join(tokens[1])
				vm := findJvmOnPath(p, dirsGlob)
				if vm != nil {
					return vm
				}
			}
		}
	}
	//find via jvm hsperfdata using regular tempdir
	path, err := hsperfdata.PerfDataPath(strconv.Itoa(int(p.Pid)))
	if err == nil {
		vm := findJvm(p, map[string]string{strconv.Itoa(int(p.Pid)): path})
		if vm != nil {
			return vm
		}
	}

	//create from process data only
	return createJvmFromProcess(p)
}

func createJvmFromProcess(p *process.Process) *jvm.JavaVm {
	cmdline, _ := p.Cmdline()
	path, _ := p.Exe()

	hostname, fqdn, _ := extruntime.GetHostname()

	vm := &jvm.JavaVm{
		Pid:           p.Pid,
		DiscoveredVia: "os-process",
		CommandLine:   cmdline,
		Path:          path,
		Hostname:      hostname,
		HostFQDN:      fqdn,
	}

	args := strings.Split(cmdline, " ")
	for i, arg := range args {
		if arg == "-cp" || arg == "-classpath" {
			vm.ClassPath = args[i+1]
			break
		}
	}

	return vm
}

func findJvmOnPath(p *process.Process, dirsGlob string) *jvm.JavaVm {
	filePaths := hsperf.FindHsPerfDataDirs(dirsGlob)
	vm := findJvm(p, filePaths)
	return vm
}

func findJvm(p *process.Process, paths map[string]string) *jvm.JavaVm {
	path := paths[strconv.Itoa(int(p.Pid))]
	if path == "" {
		return nil
	}
	return parsePerfDataBuffer(p, path)
}

func parsePerfDataBuffer(p *process.Process, path string) *jvm.JavaVm {
	tempFile := os.TempDir() + "/hsperfdata" + strconv.Itoa(int(p.Pid))
	cmd := utils.RootCommandContext(context.Background(), "cp", path, tempFile)
	err := cmd.Run()
	if err != nil {
		log.Error().Msgf("Error while copying perf data from %s to %s: %s", path, tempFile, err)
		return nil
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			log.Warn().Msgf("Error while removing temp file %s: %s", name, err)
		}
	}(tempFile)

	entryMap, err := hsperfdata.ReadPerfData(tempFile, false)
	if err != nil {
		log.Error().Msgf("Error while reading perf data from %s: %s", path, err)
		return nil
	}
	if !hsperf.IsAttachable(entryMap) {
		return nil
	}
	commandLine := hsperf.GetStringProperty(entryMap, "sun.rt.javaCommand")
	hostname, fqdn, _ := extruntime.GetHostname()
	vm := &jvm.JavaVm{
		Pid:           p.Pid,
		DiscoveredVia: "hsperfdata",
		CommandLine:   commandLine,
		MainClass:     getMainClass(commandLine),
		ClassPath:     hsperf.GetStringProperty(entryMap, "java.property.java.class.path"),
		VmArgs:        hsperf.GetStringProperty(entryMap, "java.rt.vmArgs"),
		VmName:        hsperf.GetStringProperty(entryMap, "java.vm.name"),
		VmVendor:      hsperf.GetStringProperty(entryMap, "java.vm.vendor"),
		VmVersion:     hsperf.GetStringProperty(entryMap, "java.vm.version"),
		Hostname:      hostname,
		HostFQDN:      fqdn,
	}
	uids, err := p.Uids()
	if err == nil && len(uids) > 0 {
		vm.UserId = fmt.Sprintf("%d", uids[0])
	}

	gids, err := p.Gids()
	if err == nil && len(gids) > 0 {
		vm.GroupId = fmt.Sprintf("%d", gids[0])
	}

	processPath, err := java_process.GetProcessPath(p)
	if err == nil && processPath != "" {
		vm.Path = processPath
	} else {
		exe, err := p.Exe()
		if err == nil && exe != "" {
			vm.Path = exe
		}
	}

	return vm
}

func getMainClass(commandLine string) string {
	if commandLine == "" {
		return ""
	}
	cmdLine := strings.TrimSpace(commandLine)
	firstSpace := strings.Index(commandLine, " ")
	if firstSpace > 0 {
		cmdLine = cmdLine[:firstSpace]
	}
	/*
	 * Can't use File.separator() here because the separator for the target
	 * jvm may be different than the separator for the monitoring jvm.
	 * And we also strip embedded module e.g. "module/MainClass"
	 */
	lastSlash := strings.LastIndex(cmdLine, "/")
	lastBackslash := strings.LastIndex(cmdLine, "\\")
	lastSeparator := math.Max(float64(lastSlash), float64(lastBackslash))
	if lastSeparator > 0 {
		cmdLine = cmdLine[int(lastSeparator)+1:]
	}
	lastPackageSeparator := strings.LastIndex(cmdLine, ".")
	if lastPackageSeparator > 0 {
		lastPart := cmdLine[int(lastPackageSeparator)+1:]
		/*
		 * We could have a relative path "my.module" or
		 * a module called "my.module" and a jar file called "my.jar" or
		 * class named "jar" in package "my", e.g. "my.jar".
		 * We can never be sure here, but we assume *.jar is a jar file
		 */
		if lastPart == "jar" {
			return cmdLine
		}
		return lastPart
	}
	return cmdLine
}

func isExcluded(vm *jvm.JavaVm) bool {
	if utils.ContainsPartOfString(ClasspathExcludes, vm.ClassPath) {
		log.Debug().Msgf("%s is excluded by classpath", vm.ToDebugString())
		return true
	}
	if utils.ContainsPartOfString(CommandlineExcludes, vm.CommandLine) || utils.ContainsPartOfString(CommandlineExcludes, vm.VmArgs) {
		log.Debug().Msgf("%s is excluded by command", vm.ToDebugString())
		return true
	}
	return false
}

func installSignalHandler() {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
	go func(signals <-chan os.Signal) {
		for s := range signals {
			signalName := unix.SignalName(s.(syscall.Signal))

			log.Info().Str("signal", signalName).Msg("received signal - stopping all active discoveries")
			DeactivateDataSourceDiscovery()
			DeactivateSpringDiscovery()

			switch s {
			case syscall.SIGINT:
				fmt.Println()
				os.Exit(128 + int(s.(syscall.Signal)))

			case syscall.SIGTERM:
				fmt.Printf("Terminated: %d\n", int(s.(syscall.Signal)))
				os.Exit(128 + int(s.(syscall.Signal)))
			}
		}
	}(signalChannel)
}
