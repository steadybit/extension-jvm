package jvm

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"sync"
)

var (
	ClasspathExcludes   = []string{"IntelliJ IDEA", "surefirebooter", "Eclipse"}
	CommandlineExcludes = []string{
		"JetBrains Toolbox.app",
		"IntelliJ IDEA",
		"com.intellij.idea.Main",
		"jetbrains.buildServer.agent.Launcher",
		"jetbrains.buildServer.agent.AgentMain",
		"org.jetbrains.jps.cmdline.BuildMain",
		"org.jetbrains.idea.maven.server.RemoteMavenServer",
		"org.jetbrains.jps.cmdline.Launcher",
		"org.jetbrains.plugins.scala.nailgun.NailgunRunner",
		"sun.tools.",
		"com.steadybit.javaagent.ExternalJavaagentAttachment",
		"steadybit.agent.disable-jvm-attachment",
		"-XX:+DisableAttachMechanism",
		"-XX:-EnableDynamicAgentLoading",
		"-Dcom.ibm.tools.attach.enable=no",
		"com.steadybit.agent.application.SteadybitAgentApplication",
		"SteadybitPlatformApplication",
	}
)

type javaVms struct {
	Added     <-chan JavaVm
	chAdded   chan<- JavaVm
	Removed   <-chan JavaVm
	chRemoved chan<- JavaVm
	mJvms     sync.RWMutex
	jvms      map[int32]JavaVm
}

func newJavaVms() *javaVms {
	chAdded := make(chan JavaVm)
	chRemoved := make(chan JavaVm)
	return &javaVms{
		jvms:      make(map[int32]JavaVm),
		Added:     chAdded,
		chAdded:   chAdded,
		Removed:   chRemoved,
		chRemoved: chRemoved,
	}
}

func (j *javaVms) addJvm(javaVm JavaVm) {
	if javaVm == nil || isIgnored(javaVm) {
		return
	}
	key := javaVm.Pid()

	j.mJvms.Lock()
	defer j.mJvms.Unlock()
	if _, ok := j.jvms[key]; ok {
		log.Trace().Msgf("%s already known", javaVm.ToDebugString())
		return
	}

	log.Debug().Msgf("Discovered %s", javaVm.ToDebugString())
	j.jvms[key] = javaVm
	j.chAdded <- javaVm
}

func (j *javaVms) getJvm(pid int32) JavaVm {
	for _, javaVm := range j.getJvms() {
		if javaVm.Pid() == pid {
			return javaVm
		}
	}
	return nil
}

func (j *javaVms) getJvms() []JavaVm {
	j.removeStopped()
	j.mJvms.RLock()
	defer j.mJvms.RUnlock()

	result := make([]JavaVm, 0, len(j.jvms))
	for _, javaVm := range j.jvms {
		result = append(result, javaVm)
	}
	return result
}

func (j *javaVms) removeStopped() {
	j.mJvms.Lock()
	defer j.mJvms.Unlock()

	for key, javaVm := range j.jvms {
		if javaVm.IsRunning() {
			continue
		}
		delete(j.jvms, key)
		log.Debug().Msgf("Removing %s", javaVm.ToDebugString())
		j.chRemoved <- javaVm
	}
}

func isIgnored(jvm JavaVm) bool {
	if utils.ContainsPartOfString(ClasspathExcludes, jvm.ClassPath()) {
		log.Trace().Msgf("%s is excluded by classpath", jvm.ToDebugString())
		return true
	}
	if utils.ContainsPartOfString(CommandlineExcludes, jvm.CommandLine()) || utils.ContainsPartOfString(CommandlineExcludes, jvm.VmArgs()) {
		log.Trace().Msgf("%s is excluded by command", jvm.ToDebugString())
		return true
	}
	return false
}
