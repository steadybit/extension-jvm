/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import com.steadybit.agent.commons.AgentUtils;
import com.steadybit.agent.commons.ProcFs;
import com.steadybit.cri.CriClient;
import com.steadybit.docker.DockerClient;
import com.steadybit.javaagent.attachment.JavaVm;
import org.apache.commons.lang3.SystemUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import oshi.software.os.OSProcess;

import javax.annotation.PostConstruct;
import javax.annotation.PreDestroy;
import java.io.IOException;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.Arrays;
import java.util.Collection;
import java.util.Collections;
import java.util.List;
import java.util.Optional;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;

/**
 * This class is used to find all JVMs runnng on the host (regardless of running inside a container or not).
 * <p>
 * The JVMs are either discovered by their hsperf-file or by listing all `java` processes.
 * <p>
 * The hsperf-files are read to obtain additional data from the java processes.
 */
public class DefaultJavaVms extends BaseJavaVms {
    private static final Logger log = LoggerFactory.getLogger(DefaultJavaVms.class);
    private static final List<String> CLASSPATH_EXCLUDES = Arrays.asList("IntelliJ IDEA", "surefirebooter");
    private static final List<String> COMMANDLINE_EXCLUDES = Arrays.asList("com.intellij.idea.Main", "jetbrains.buildServer.agent.Launcher",
            "jetbrains.buildServer.agent.AgentMain", "org.jetbrains.jps.cmdline.BuildMain", "org.jetbrains.idea.maven.server.RemoteMavenServer",
            "org.jetbrains.jps.cmdline.Launcher", "org.jetbrains.plugins.scala.nailgun.NailgunRunner", "sun.tools.",
            "com.steadybit.javaagent.ExternalJavaagentAttachment", "steadybit.agent.disable-jvm-attachment",
            "-XX:+DisableAttachMechanism", "-Dcom.ibm.tools.attach.enable=no"
    );
    private final Set<Integer> pidExcludes = Collections.newSetFromMap(new ConcurrentHashMap<>());
    private final HotspotJvmWatcher hotspotJvmWatcher;
    private final JavaProcessWatcher javaProcessWatcher;
    private final DockerClient dockerClient;
    private final CriClient criClient;
    private final HotspotJvmWatcher.Listener hotspotListener;
    private final JavaProcessWatcher.Listener processListener;

    public DefaultJavaVms(HotspotJvmWatcher hotspotJvmWatcher, JavaProcessWatcher javaProcessWatcher, DockerClient dockerClient, CriClient criClient) {
        this.hotspotJvmWatcher = hotspotJvmWatcher;
        this.javaProcessWatcher = javaProcessWatcher;
        this.dockerClient = dockerClient;
        this.criClient = criClient;
        this.processListener = p -> {
            var pid = p.getProcessID();
            if (!this.jvms.containsKey(pid) && !this.pidExcludes.contains(pid)) {
                this.addJvm(this.createJvm(p));
            }
        };
        this.hotspotListener = pid -> {
            if (!this.jvms.containsKey(pid) && !this.pidExcludes.contains(pid) && this.javaProcessWatcher.isRunning(pid)) {
                this.addJvm(this.hotspotJvmWatcher.getJvm(pid));
            }
        };
    }

    @PostConstruct
    public void activate() {
        System.setProperty("sun.jvmstat.perdata.syncWaitMs", "0");
        log.debug("Adding steadybit Agent PID {} to JVM discovery excludes.", AgentUtils.getAgentPid());
        this.pidExcludes.add(AgentUtils.getAgentPid());
        this.javaProcessWatcher.addListener(this.processListener);
        this.hotspotJvmWatcher.addListener(this.hotspotListener);
    }

    @PreDestroy
    public void deactivate() {
        this.javaProcessWatcher.removeListener(this.processListener);
        this.hotspotJvmWatcher.removeListener(this.hotspotListener);
    }

    @Override
    public Collection<JavaVm> getJavaVms() {
        this.removeStoppedJvms();
        return super.getJavaVms();
    }

    @Override
    public Optional<JavaVm> getJavaVm(int pid) {
        if (this.javaProcessWatcher.isRunning(pid)) {
            return super.getJavaVm(pid);
        }
        return Optional.empty();
    }

    private void removeStoppedJvms() {
        for (var integer : this.jvms.keySet()) {
            if (!this.javaProcessWatcher.isRunning(integer) && !Thread.currentThread().isInterrupted()) {
                //if the current thread is interrupted oshi will report null for the process which is incorrect.
                //when the discovery is running into a timeout, this is most of the case.
                //so we only remove the process if we weren't interrupted
                log.debug("JVM process {} not present. Removing from VM List.", integer);
                this.removeJvm(integer);
            }
        }
    }

    private JavaVm createJvm(OSProcess process) {
        var containerId = this.getContainerIdForProcess(process.getProcessID());
        if (containerId == null) {
            return this.createHostJvm(process);
        }

        var containerFs = ProcFs.ROOT.getProcessRoot(process.getProcessID());
        var containerPid = this.getContainerPid(process.getProcessID(), containerFs);
        if (containerPid != null) {
            return this.createContainerizedJvm(process, containerId, containerPid, containerFs);
        }
        return null;
    }

    private JavaVm createHostJvm(OSProcess process) {
        JavaVm jvm;

        //find via jvm hsperfdata using a chroot fs
        if (SystemUtils.IS_OS_UNIX) {
            var rootFs = ProcFs.ROOT.getProcessRoot(process.getProcessID());
            jvm = this.hotspotJvmWatcher.getJvmFromRoot(process.getProcessID(), process.getProcessID(), rootFs);
            if (jvm != null) {
                return jvm;
            }
        }

        //find via jvm hsperfdata using an alternative tempdir
        var commandLine = process.getCommandLine();
        if (commandLine != null) {
            for (var arg : commandLine.split("\0")) {
                if (arg.startsWith("-Djava.io.tmpdir")) {
                    var tokens = arg.split("=");
                    if (tokens.length > 1) {
                        jvm = this.hotspotJvmWatcher.getJvmFromHsPerfDataDir(process.getProcessID(), Paths.get(tokens[1]));
                        if (jvm != null) {
                            return jvm;
                        }
                    }
                }
            }
        }

        //find via jvm hsperfdata using regular tempdir
        jvm = this.hotspotJvmWatcher.getJvm(process.getProcessID());
        if (jvm != null) {
            return jvm;
        }

        //create from process data only
        return this.getJvmFromProcess(process);
    }

    private JavaVm createContainerizedJvm(OSProcess process, String containerId, Integer containerPid, Path containerFs) {
        var vm = this.hotspotJvmWatcher.getJvmFromRoot(containerPid, process.getProcessID(), containerFs);
        if (vm == null) {
            vm = this.getJvmFromProcess(process);
        }
        vm.setInContainerPid(containerPid);
        vm.setContainerId(containerId);
        return vm;
    }

    private JavaVm getJvmFromProcess(OSProcess process) {
        var commandStr = process.getCommandLine();
        var vm = new JavaVm(process.getProcessID(), "os-process");
        vm.setPath(process.getPath());
        vm.setCommandLine(commandStr.replace('\0', ' '));

        for (var arg : commandStr.split("\0")) {
            if ("-cp".equals(arg) || "-classpath".equals(arg)) {
                vm.setClasspath(arg);
                break;
            }
        }

        vm.setUserId(process.getUserID());
        vm.setGroupId(process.getGroupID());
        return vm;
    }

    private Integer getContainerPid(int hostPid, Path containerFs) {
        log.trace("Looking for containerized JVM {} in {}", hostPid, containerFs);

        try {
            var pid = ProcFs.ROOT.findNamespacePid(hostPid);
            if (pid != null) {
                log.trace("Found Host PID {} is {} in container via proc/status", hostPid, pid);
                return pid;
            }
        } catch (IOException ex) {
            log.trace("Failed to read Container PID from file: {}", ex.getMessage());
        }

        var containerPids = this.hotspotJvmWatcher.getJvmPids(containerFs);
        if (!containerPids.isEmpty()) {
            log.trace("Potential container PIDs found for JVM {}: {} ", hostPid, containerPids);
            for (var containerPid : containerPids) {
                try {
                    var pid = ProcFs.withRoot(containerFs).readPidFromSchedulerDebug(containerPid);
                    if (pid != null && pid == hostPid) {
                        log.trace("Found Host PID {} is {} in container via proc/sched", hostPid, pid);
                        return containerPid;
                    }
                } catch (IOException ex) {
                    log.trace("Failed to match hostPid {} with containerPid {} using sched: {}", hostPid, containerPid, ex.getMessage());
                }
            }
        }

        log.debug("Could not get Container PID for {}", hostPid);
        return null;
    }

    private boolean isExcluded(JavaVm vm) {
        if (CLASSPATH_EXCLUDES.stream().anyMatch(bcp -> vm.getClasspath() != null && vm.getClasspath().contains(bcp))) {
            log.debug("{} is excluded by classpath", vm);
            return true;
        }

        if (COMMANDLINE_EXCLUDES.stream().anyMatch(bcl -> vm.getCommandLine() != null && vm.getCommandLine().contains(bcl))) {
            log.debug("{} is excluded by command", vm);
            return true;
        }
        return false;
    }

    @Override
    protected void addJvm(JavaVm jvm) {
        if (jvm == null || this.isExcluded(jvm)) {
            return;
        }

        if (log.isDebugEnabled()) {
            log.debug("Discovered JVM {}", jvm.toDebugString());
        }
        super.addJvm(jvm);
    }

    private String getContainerIdForProcess(int pid) {
        var containerId = this.dockerClient.getContainerIdForProcess(String.valueOf(pid));
        if (containerId != null) {
            return containerId;
        }
        return this.criClient.getContainerIdForProcess(String.valueOf(pid));
    }
}