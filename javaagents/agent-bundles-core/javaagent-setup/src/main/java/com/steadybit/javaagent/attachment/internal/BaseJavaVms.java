/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import com.steadybit.javaagent.attachment.JavaVm;
import com.steadybit.javaagent.attachment.JavaVms;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.Collection;
import java.util.List;
import java.util.Map;
import java.util.Optional;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.CopyOnWriteArrayList;

public class BaseJavaVms implements JavaVms {
    private static final Logger log = LoggerFactory.getLogger(BaseJavaVms.class);
    protected final Map<Integer, JavaVm> jvms = new ConcurrentHashMap<>();
    private final List<Listener> listeners = new CopyOnWriteArrayList<>();

    protected void addJvm(JavaVm jvm) {
        this.jvms.put(jvm.getPid(), jvm);
        for (var listener : this.listeners) {
            try {
                listener.addedJvm(jvm);
            } catch (Exception e) {
                log.warn("Error in listener for addedJvm {}", listener, e);
            }
        }
    }

    protected void removeJvm(int pid) {
        var jvm = this.jvms.remove(pid);
        if (jvm == null) {
            return;
        }

        for (var listener : this.listeners) {
            try {
                listener.removedJvm(jvm);
            } catch (Exception e) {
                log.warn("Error in listener for removedJvm {}", listener, e);
            }
        }
    }

    @Override
    public Collection<JavaVm> getJavaVms() {
        return this.jvms.values();
    }

    @Override
    public Optional<JavaVm> getJavaVm(int pid) {
        return Optional.ofNullable(this.jvms.get(pid));
    }

    @Override
    public void addListener(Listener l) {
        this.listeners.add(l);
        for (var jvm : this.getJavaVms()) {
            try {
                l.addedJvm(jvm);
            } catch (Exception e) {
                log.warn("Error in listener for addedJvm {}", l, e);
            }
        }
    }

    @Override
    public void removeListener(Listener l) {
        this.listeners.remove(l);
    }
}
