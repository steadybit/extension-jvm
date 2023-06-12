/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import static org.assertj.core.api.Assertions.assertThat;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;
import oshi.software.os.OSProcess;
import oshi.software.os.OperatingSystem;

import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

class JavaProcessWatcherTest {
    private final OperatingSystem os = mock(OperatingSystem.class);
    private final Map<Integer, OSProcess> processMocks = new ConcurrentHashMap<>();
    private final List<OSProcess> processes = new ArrayList<>();
    private final JavaProcessWatcher watcher = new JavaProcessWatcher(this.os);

    @BeforeEach
    void setUp() {
        this.watcher.addListener(this.processes::add);
        when(this.os.getProcesses()).then(i -> new ArrayList<>(this.processMocks.values()));
    }

    @Test
    void should_report_new_process() {
        var existsBefore = this.startMockProcess(1000, "java", OSProcess.State.RUNNING);
        var other = this.startMockProcess(9000, "other", OSProcess.State.RUNNING);
        var stopped = this.startMockProcess(9001, "java", OSProcess.State.STOPPED);

        this.watcher.updatePids();
        assertThat(this.processes).contains(existsBefore).doesNotContain(other, stopped);

        var started1 = this.startMockProcess(1001, "java", OSProcess.State.RUNNING);
        var started2 = this.startMockProcess(1002, "java", OSProcess.State.RUNNING);
        this.watcher.updatePids();
        assertThat(this.processes).contains(started1, started2);
    }

    @Test
    void should_report_reused_pid() throws InterruptedException {
        var process = this.startMockProcess(1001, "java", OSProcess.State.RUNNING);
        this.watcher.updatePids();
        assertThat(this.processes).contains(process);

        this.stopMockProcess(1001);
        this.watcher.updatePids();

        var reusedPid = this.startMockProcess(1001, "java", OSProcess.State.RUNNING);
        this.watcher.updatePids();
        assertThat(this.processes).contains(reusedPid);
    }

    private void stopMockProcess(int pid) {
        when(this.os.getProcess(pid)).thenReturn(null);
        this.processMocks.remove(pid);
    }

    private OSProcess startMockProcess(int pid, String name, OSProcess.State state) {
        var process = mock(OSProcess.class);
        when(process.getProcessID()).thenReturn(pid);
        when(process.getName()).thenReturn(name);
        when(process.getState()).thenReturn(state);
        when(this.os.getProcess(pid)).thenReturn(process);
        this.processMocks.put(pid, process);
        return process;
    }
}