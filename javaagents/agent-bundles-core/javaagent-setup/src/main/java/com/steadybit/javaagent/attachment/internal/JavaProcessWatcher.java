/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import com.steadybit.agent.system.SystemInfo;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.scheduling.annotation.Scheduled;
import oshi.software.os.OSProcess;
import oshi.software.os.OperatingSystem;

import java.util.BitSet;
import java.util.EnumSet;
import java.util.List;
import java.util.Set;
import java.util.concurrent.CopyOnWriteArrayList;

public class JavaProcessWatcher {
    private static final int MAX_PID = 99999;
    private static final Set<OSProcess.State> RUNNING_STATES = EnumSet.of(OSProcess.State.RUNNING, OSProcess.State.WAITING, OSProcess.State.SLEEPING);
    private static final Logger log = LoggerFactory.getLogger(JavaProcessWatcher.class);
    private final OperatingSystem os;
    private final List<Listener> listeners = new CopyOnWriteArrayList<>();
    private BitSet pids = new BitSet(MAX_PID);

    public JavaProcessWatcher() {
        this(SystemInfo.getOperatingSystem());
    }

    JavaProcessWatcher(OperatingSystem os) {
        this.os = os;
    }

    @Scheduled(fixedDelay = 5_000L)
    public void updatePids() {
        try {
            var newPids = new BitSet(MAX_PID);

            for (var process : JavaProcessWatcher.this.os.getProcesses()) {
                var pid = process.getProcessID();
                newPids.set(pid);
                if (!this.pids.get(pid)) {
                    JavaProcessWatcher.this.newProcess(process);
                }
            }

            this.pids = newPids;
        } catch (Throwable e) {
            log.warn("Error while looking for new java processes", e);
        }
    }

    private boolean isRunning(OSProcess process) {
        return process != null && RUNNING_STATES.contains(process.getState());
    }

    private void newProcess(OSProcess process) {
        if (this.isJava(process)) {
            log.trace("Discovered new java process: {}", process.getProcessID());
            for (var listener : this.listeners) {
                try {
                    listener.newProcess(process);
                } catch (Exception e) {
                    log.warn("Error in listener for newProcess {}", listener, e);
                }
            }
        }
    }

    private boolean isJava(OSProcess process) {
        return this.isRunning(process) && process.getName().endsWith("java");
    }

    public boolean isRunning(int pid) {
        return this.isRunning(this.os.getProcess(pid));
    }

    public void addListener(Listener l) {
        this.listeners.add(l);
        this.pids.stream().forEach(pid -> {
            var process = this.os.getProcess(pid);
            if (this.isJava(process)) {
                try {
                    l.newProcess(process);
                } catch (Exception e) {
                    log.warn("Error in listener for newProcess {}", l, e);
                }
            }
        });
    }

    public void removeListener(Listener l) {
        this.listeners.remove(l);
    }

    public interface Listener {
        void newProcess(OSProcess process);
    }
}
