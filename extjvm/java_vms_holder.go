package extjvm

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/process"
	"github.com/steadybit/extension-jvm/extjvm/java_process"
  "github.com/steadybit/extension-jvm/extjvm/utils"
  "github.com/steadybit/extension-kit/extutil"
	"github.com/xin053/hsperfdata"
	"math"
	"os"
	"path/filepath"
	"runtime"
  "strconv"
  "strings"
	"sync"
)

var (
	pidExludes          []int32
	jvms                sync.Map //map[int32]java_process.JavaVm
	ClasspathExcludes   = []string{"IntelliJ IDEA", "surefirebooter"}
	CommandlineExcludes = []string{"IntelliJ IDEA", "com.intellij.idea.Main", "jetbrains.buildServer.agent.Launcher",
		"jetbrains.buildServer.agent.AgentMain", "org.jetbrains.jps.cmdline.BuildMain", "org.jetbrains.idea.maven.server.RemoteMavenServer",
		"org.jetbrains.jps.cmdline.Launcher", "org.jetbrains.plugins.scala.nailgun.NailgunRunner", "sun.tools.",
		"com.steadybit.javaagent.ExternalJavaagentAttachment", "steadybit.agent.disable-jvm-attachment",
		"-XX:+DisableAttachMechanism", "-Dcom.ibm.tools.attach.enable=no"}
)

type JavaVMS struct{}

type JavaVm struct {
	Pid           int32
	CommandLine   string
	MainClass     string
	ClassPath     string
	ContainerId   string
	InContainerId int
	VmVersion     string
	VmVendor      string
	VmName        string
	VmArgs        string
	UserId        string
	GroupId       string
	Path          string
	DiscoveredVia string
}

func Activate(agentPid int) {
	pidExludes = append(pidExludes, extutil.ToInt32(agentPid))

	java_process.AddListener(&JavaVMS{})
}

func (j *JavaVMS) NewProcess(p *process.Process) {
	if !utils.Contains(pidExludes, p.Pid) {
		_, ok := jvms.Load(p.Pid)
		if !ok {
			addJvm(createJvm(p))
		}
	}
}

func addJvm(jvm *JavaVm) {
	jvms.Store(jvm.Pid, jvm)
}
func createJvm(process *process.Process) *JavaVm {
	containerId := getContainerIdForProcess(process)
	if containerId == "" {
		return createHostJvm(process)
	}
	//var containerFs = ProcFs.ROOT.getProcessRoot(process.getProcessID());
	//var containerPid = this.getContainerPid(process.getProcessID(), containerFs);
	//if (containerPid != null) {
	//  return this.createContainerizedJvm(process, containerId, containerPid, containerFs);
	//}
	return nil
}

func createHostJvm(p *process.Process) *JavaVm {
	if runtime.GOOS != "windows" {
		rootPath := fmt.Sprintf("/proc/%d/root", p.Pid)
		dirsGlob := filepath.Join(rootPath, os.TempDir(), "hsperfdata_*", strconv.Itoa(int(p.Pid)))
		jvm := findJvmOnPath(p, dirsGlob)
		if jvm != nil {
			return jvm
		}
	}

	//find via jvm hsperfdata using an alternative tempdir
	cmdline, err := p.Cmdline()
	if err == nil && cmdline != "" {
		arg := strings.Split(cmdline, " ")
		if strings.HasPrefix(arg[0], "-Djava.io.tmpdir") {
			tokens := strings.Split(arg[0], "=")
			if len(tokens) > 1 {
				dirsGlob := filepath.Join(tokens[1], "hsperfdata_*", strconv.Itoa(int(p.Pid)))
				jvm := findJvmOnPath(p, dirsGlob)
				if jvm != nil {
					return jvm
				}
			}
		}
	}
	//find via jvm hsperfdata using regular tempdir
	path, err := hsperfdata.PerfDataPath(strconv.Itoa(int(p.Pid)))
	if err == nil {
		jvm := findJvm(p, map[string]string{strconv.Itoa(int(p.Pid)): path})
		if jvm != nil {
			return jvm
		}
	}

	//create from process data only
	return createJvmFromProcess(p)
}

func createJvmFromProcess(p *process.Process) *JavaVm {
	cmdline, _ := p.Cmdline()
	path, _ := p.Exe()

	jvm := &JavaVm{
		Pid:           p.Pid,
		DiscoveredVia: "os-process",
		CommandLine:   cmdline,
		Path:          path,
	}

	args := strings.Split(cmdline, " ")
	for i, arg := range args {
		if arg == "-cp" || arg == "-classpath" {
			jvm.ClassPath = args[i+1]
			break
		}
	}

	return jvm
}

func findJvmOnPath(p *process.Process, dirsGlob string) *JavaVm {
	paths, err := filepath.Glob(dirsGlob)
	if err != nil {
		log.Error().Msgf("Error while globbing %s: %s", dirsGlob, err)
		return nil
	}

	filePaths := make(map[string]string)
	for _, path := range paths {
		pid := filepath.Base(path)
		filePaths[pid] = path
	}
	jvm := findJvm(p, filePaths)
	return jvm
}

func findJvm(p *process.Process, paths map[string]string) *JavaVm {
	return parsePerfDataBuffer(p, paths[strconv.Itoa(int(p.Pid))])
}

func parsePerfDataBuffer(p *process.Process, path string) *JavaVm {
	entryMap, err := hsperfdata.ReadPerfData(path, false)
	if err != nil {
		log.Error().Msgf("Error while reading perf data from %s: %s", path, err)
		return nil
	}
	commandLine := getStringProperty(entryMap, "sun.rt.javaCommand")
	jvm := &JavaVm{
		Pid:           p.Pid,
		DiscoveredVia: "hsperfdata",
		CommandLine:   commandLine,
		MainClass:     getMainClass(commandLine),
		ClassPath:     getStringProperty(entryMap, "java.property.java.class.path"),
		VmArgs:        getStringProperty(entryMap, "java.rt.vmArgs"),
		VmName:        getStringProperty(entryMap, "java.vm.name"),
		VmVendor:      getStringProperty(entryMap, "java.vm.vendor"),
		VmVersion:     getStringProperty(entryMap, "java.vm.version"),
	}
	uids, err := p.Uids()
	if err == nil && len(uids) > 0 {
		jvm.UserId = fmt.Sprintf("%d", uids[0])
	}

	gids, err := p.Gids()
	if err == nil && len(gids) > 0 {
		jvm.GroupId = fmt.Sprintf("%d", gids[0])
	}

	exe, err := p.Exe()
	if err == nil && exe != "" {
		jvm.Path = exe
	}

	return jvm
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

func getStringProperty(entryMap map[string]interface{}, key string) string {
	if value, ok := entryMap[key]; ok {
		return value.(string)
	}
	log.Error().Msgf("Could not get property %s from perfdata", key)
	return ""
}

func getContainerIdForProcess(process *process.Process) string {
	//TODO: implement
	return ""
}

func isExcluded(vm JavaVm) bool {
	if utils.ContainsPartOfString(ClasspathExcludes, vm.ClassPath) {
		log.Debug().Msgf("%+v is excluded by classpath", vm)
		return true
	}
	if utils.ContainsPartOfString(CommandlineExcludes, vm.ClassPath) {
		log.Debug().Msgf("%+v is excluded by command", vm)
		return true
	}
	return false
}
