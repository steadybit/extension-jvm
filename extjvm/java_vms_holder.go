package extjvm

import (
  "fmt"
  "github.com/rs/zerolog/log"
  "github.com/shirou/gopsutil/process"
  "github.com/steadybit/extension-jvm/extjvm/hotspot"
  "github.com/steadybit/extension-jvm/extjvm/hsperf"
  "github.com/steadybit/extension-jvm/extjvm/java_process"
  "github.com/steadybit/extension-jvm/extjvm/jvm"
  "github.com/steadybit/extension-jvm/extjvm/procfs"
  "github.com/steadybit/extension-jvm/extjvm/utils"
  "github.com/steadybit/extension-kit/extutil"
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
  pidExludes          []int32
  jvms                sync.Map //map[int32]java_process.JavaVm
  ClasspathExcludes   = []string{"IntelliJ IDEA", "surefirebooter"}
  CommandlineExcludes = []string{"IntelliJ IDEA", "com.intellij.idea.Main", "jetbrains.buildServer.agent.Launcher",
    "jetbrains.buildServer.agent.AgentMain", "org.jetbrains.jps.cmdline.BuildMain", "org.jetbrains.idea.maven.server.RemoteMavenServer",
    "org.jetbrains.jps.cmdline.Launcher", "org.jetbrains.plugins.scala.nailgun.NailgunRunner", "sun.tools.",
    "com.steadybit.javaagent.ExternalJavaagentAttachment", "steadybit.agent.disable-jvm-attachment",
    "-XX:+DisableAttachMechanism", "-Dcom.ibm.tools.attach.enable=no"}
  listeners []Listener
)

type Listener interface {
  AddedJvm(jvm *jvm.JavaVm)
  RemovedJvm(jvm *jvm.JavaVm)
}

type JavaVMS struct{}

func AddListener(listener Listener) {
  listeners = append(listeners, listener)
  for _, jvm := range GetJVMs() {
    listener.AddedJvm(&jvm)
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

func Activate(agentPid int) {
  pidExludes = append(pidExludes, extutil.ToInt32(agentPid))

  java_process.AddListener(&JavaVMS{})
  hotspot.AddListener(&JavaVMS{})
}

func (j *JavaVMS) NewProcess(p *process.Process) {
  if !utils.Contains(pidExludes, p.Pid) {
    _, ok := jvms.Load(p.Pid)
    if !ok {
      addJvm(createJvm(p))
    }
  }
}

func (j *JavaVMS) NewHotspotProcess(p *process.Process) {
  if !utils.Contains(pidExludes, p.Pid) {
    _, ok := jvms.Load(p.Pid)
    if !ok && java_process.IsRunning(p) {
      addJvm(createJvm(p))
    }
  }
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
    jvm := value.(jvm.JavaVm)
    p, err := process.NewProcess(jvm.Pid)
    if err != nil {
      log.Warn().Err(err).Msg("Error in listener for newProcess")
      return true
    }
    if !java_process.IsRunning(p) {
      jvms.Delete(key)
      for _, listener := range listeners {
        listener.RemovedJvm(&jvm)
      }
    }
    return true
  })
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
  filePaths := hotspot.GetRootHsPerfPaths(p.Pid, containerFs)
  if len(filePaths) == 0 {
    log.Error().Msgf("Could not find hsperfdata root path for container %s", containerId)
    return nil
  }
  hsPerfDataPath := filePaths[containerId]
  if hsPerfDataPath == "" {
    log.Error().Msgf("Could not find hsperfdata path for container %s", containerId)
    return nil
  }
  javaVm := parsePerfDataBuffer(p, hsPerfDataPath)
  if javaVm == nil {
    log.Error().Msgf("Could not parse hsperfdata for container %s", containerId)
    return nil
  }
  javaVm.InContainerPid = int(containerPid)
  javaVm.ContainerId = containerId
  return javaVm
}

func createHostJvm(p *process.Process) *jvm.JavaVm {
  if runtime.GOOS != "windows" {
    rootPath := procfs.GetProcessRoot(p.Pid)
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

func createJvmFromProcess(p *process.Process) *jvm.JavaVm {
  cmdline, _ := p.Cmdline()
  path, _ := p.Exe()

  jvm := &jvm.JavaVm{
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

func findJvmOnPath(p *process.Process, dirsGlob string) *jvm.JavaVm {
  filePaths := hsperf.FindHsPerfDataDirs(dirsGlob)
  jvm := findJvm(p, filePaths)
  return jvm
}

func findJvm(p *process.Process, paths map[string]string) *jvm.JavaVm {
  path := paths[strconv.Itoa(int(p.Pid))]
  if path == "" {
    return nil
  }
  return parsePerfDataBuffer(p, path)
}

func parsePerfDataBuffer(p *process.Process, path string) *jvm.JavaVm {
  entryMap, err := hsperfdata.ReadPerfData(path, false)
  if err != nil {
    log.Error().Msgf("Error while reading perf data from %s: %s", path, err)
    return nil
  }
  if !hsperf.IsAttachable(entryMap) {
    return nil
  }
  commandLine := hsperf.GetStringProperty(entryMap, "sun.rt.javaCommand")
  jvm := &jvm.JavaVm{
    Pid:           p.Pid,
    DiscoveredVia: "hsperfdata",
    CommandLine:   commandLine,
    MainClass:     getMainClass(commandLine),
    ClassPath:     hsperf.GetStringProperty(entryMap, "java.property.java.class.path"),
    VmArgs:        hsperf.GetStringProperty(entryMap, "java.rt.vmArgs"),
    VmName:        hsperf.GetStringProperty(entryMap, "java.vm.name"),
    VmVendor:      hsperf.GetStringProperty(entryMap, "java.vm.vendor"),
    VmVersion:     hsperf.GetStringProperty(entryMap, "java.vm.version"),
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

func isExcluded(vm *jvm.JavaVm) bool {
  if utils.ContainsPartOfString(ClasspathExcludes, vm.ClassPath) {
    log.Debug().Msgf("%s is excluded by classpath", vm.ToDebugString())
    return true
  }
  if utils.ContainsPartOfString(CommandlineExcludes, vm.ClassPath) {
    log.Debug().Msgf("%s is excluded by command", vm.ToDebugString())
    return true
  }
  return false
}

func InstallSignalHandler() {
  signalChannel := make(chan os.Signal, 1)
  signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
  go func(signals <-chan os.Signal) {
    for s := range signals {
      signalName := unix.SignalName(s.(syscall.Signal))

      log.Debug().Str("signal", signalName).Msg("received signal - stopping all active discoveries")
      jvms.Range(func(key, value interface{}) bool {
        //jvm := value.(*jvm.JavaVm)
        // TODO: do something with the jvm
        return true
      })


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
