/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal.jvmattach;

import com.steadybit.agent.commons.AgentUtils;
import com.steadybit.agent.system.SystemInfo;
import com.steadybit.javaagent.attachment.JavaVm;
import com.steadybit.javaagent.attachment.internal.JavaProcessWatcher;
import lombok.SneakyThrows;
import static org.assertj.core.api.Assertions.assertThat;
import org.junit.jupiter.api.Test;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.ArgumentMatchers.anyLong;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;
import oshi.software.os.OSProcess;

import java.io.File;
import java.io.IOException;
import java.util.concurrent.atomic.AtomicReference;

class HostVmAttachmentTest {
    private static final File AGENT_JAR = new File("target/javaagent/javaagent-main.jar");
    private static final File INIT_JAR = new File("target/javaagent/javaagent-init.jar");
    private final JavaProcessWatcher watcher = mock(JavaProcessWatcher.class);
    private final OSProcess agent = SystemInfo.getOperatingSystem().getProcess(AgentUtils.getAgentPid());

    @Test
    void should_run_external_attach_with_same_user() throws IOException {
        var attachCommand = new AtomicReference<String[]>();
        var attachment = new HostVmAttachment(this.mockJvm(1234, this.agent.getUserID(), this.agent.getGroupID()), this.watcher) {
            @SneakyThrows
            @Override
            protected Process startProcess(String[] cmd) {
                attachCommand.set(cmd);
                var process = mock(Process.class);
                when(process.waitFor(anyLong(), any())).thenReturn(true);
                when(process.exitValue()).thenReturn(0);
                return process;
            }
        };

        var attach = attachment.attach(AGENT_JAR, INIT_JAR, 42899);
        assertThat(attach).isTrue();

        assertThat(attachCommand.get()).containsExactly("java",
                "-Xms16m",
                "-Xmx16m",
                "-XX:+UseSerialGC",
                "-XX:+PerfDisableSharedMem",
                "-Dsun.tools.attach.attachTimeout=30000",
                "-Dsteadybit.agent.disable-jvm-attachment",
                "-jar",
                INIT_JAR.getAbsolutePath(),
                "pid=1234",
                "hostpid=1234",
                "host=127.0.0.1",
                "port=42899",
                "agentJar=" + AGENT_JAR.getAbsolutePath());
    }

    @Test
    void should_run_external_attach_with_different_user() throws IOException {
        var attachCommand = new AtomicReference<String[]>();
        var attachment = new HostVmAttachment(this.mockJvm(1234, "9999", "9999"), this.watcher) {
            @SneakyThrows
            @Override
            protected Process startProcess(String[] cmd) {
                attachCommand.set(cmd);
                var process = mock(Process.class);
                when(process.waitFor(anyLong(), any())).thenReturn(true);
                when(process.exitValue()).thenReturn(0);
                return process;
            }
        };

        var attach = attachment.attach(AGENT_JAR, INIT_JAR, 42899);
        assertThat(attach).isTrue();

        assertThat(attachCommand.get()).containsExactly("java",
                "-Xms16m",
                "-Xmx16m",
                "-XX:+UseSerialGC",
                "-XX:+PerfDisableSharedMem",
                "-Dsun.tools.attach.attachTimeout=30000",
                "-Dsteadybit.agent.disable-jvm-attachment",
                "-jar",
                INIT_JAR.getAbsolutePath(),
                "pid=1234",
                "hostpid=1234",
                "host=127.0.0.1",
                "port=42899",
                "agentJar=" + AGENT_JAR.getAbsolutePath(),
                "uid=9999",
                "gid=9999");
    }

    private JavaVm mockJvm(int pid, String userID, String groupID) {
        when(this.watcher.isRunning(pid)).thenReturn(true);
        var javaVm = new JavaVm(pid);
        javaVm.setUserId(userID);
        javaVm.setGroupId(groupID);
        return javaVm;
    }
}