/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import com.steadybit.javaagent.attachment.JavaVm;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.scheduling.annotation.Scheduled;

import java.nio.file.Path;
import java.util.Collection;
import java.util.Collections;
import java.util.List;
import java.util.Set;
import java.util.concurrent.CopyOnWriteArrayList;

public class HotspotJvmWatcher {
    private static final Logger log = LoggerFactory.getLogger(HotspotJvmWatcher.class);
    private final List<Listener> listeners = new CopyOnWriteArrayList<>();
    private final HotspotJvmHelper helper = new HotspotJvmHelper();
    private Set<Integer> pids = Collections.emptySet();

    @Scheduled(fixedDelay = 5_000L)
    public void updatePids() {
        try {
            var newPids = this.helper.getJvmPids();
            for (var pid : newPids) {
                if (!this.pids.contains(pid)) {
                    this.newProcess(pid);
                }
            }
            this.pids = newPids;
        } catch (Throwable e) {
            log.warn("Error while looking for new hotspot jvms", e);
        }
    }

    private void newProcess(int pid) {
        log.trace("Discovered new java process: {}", pid);
        for (var listener : this.listeners) {
            try {
                listener.newProcess(pid);
            } catch (Exception e) {
                log.warn("Error in listener for newProcess {}", listener, e);
            }
        }
    }

    public Collection<Integer> getJvmPids(Path rootFs) {
        return this.helper.getJvmPids(rootFs);
    }

    public JavaVm getJvm(int hostPid) {
        return this.helper.getJvm(hostPid);
    }

    public JavaVm getJvmFromRoot(int pid, int hostPid, Path rootFs) {
        return this.helper.getJvmFromRoot(pid, hostPid, rootFs);
    }

    public JavaVm getJvmFromHsPerfDataDir(int pid, Path temp) {
        return this.helper.getJvmFromHsPerfDataDir(pid, temp);
    }

    public void addListener(Listener l) {
        this.listeners.add(l);
        for (var pid : this.pids) {
            try {
                l.newProcess(pid);
            } catch (Exception e) {
                log.warn("Error in listener for newProcess {}", l, e);
            }
        }
    }

    public void removeListener(Listener l) {
        this.listeners.remove(l);
    }

    public interface Listener {
        void newProcess(int pid);
    }
}
