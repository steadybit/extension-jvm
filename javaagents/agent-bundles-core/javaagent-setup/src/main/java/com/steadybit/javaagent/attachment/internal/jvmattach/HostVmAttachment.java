/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal.jvmattach;

import com.steadybit.agent.commons.AgentUtils;
import com.steadybit.agent.system.SystemInfo;
import com.steadybit.javaagent.attachment.JavaVm;
import com.steadybit.javaagent.attachment.JvmAttachmentException;
import com.steadybit.javaagent.attachment.internal.JavaProcessWatcher;
import org.apache.commons.io.IOUtils;
import org.apache.commons.lang3.ArrayUtils;
import org.apache.commons.lang3.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.nio.charset.Charset;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.Map;
import java.util.concurrent.TimeUnit;

public class HostVmAttachment implements JvmAttachment {
    private static final Logger log = LoggerFactory.getLogger(HostVmAttachment.class);
    private static final long ATTACH_PROCESS_TIMEOUT_SECONDS = 60;
    private final JavaProcessWatcher javaProcessWatcher;
    private final JavaVm vm;

    public HostVmAttachment(JavaVm vm, JavaProcessWatcher javaProcessWatcher) {
        this.javaProcessWatcher = javaProcessWatcher;
        this.vm = vm;
    }

    @Override
    public boolean attach(File agentJar, File initJar, int agentHttpPort) {
        if (!this.javaProcessWatcher.isRunning(this.vm.getPid())) {
            log.debug("Process not running. Skipping attachment to JVM {}", this.vm);
            return false;
        }

        return this.externalAttach(agentJar, initJar, agentHttpPort);
    }

    private boolean externalAttach(File agentJar, File initJar, int agentHttpPort) {
        var attachCommand = new String[] {
                this.getJavaExecutable(),
                "-Xms16m",
                "-Xmx16m",
                "-XX:+UseSerialGC",
                "-XX:+PerfDisableSharedMem",
                "-Dsun.tools.attach.attachTimeout=30000",
                "-Dsteadybit.agent.disable-jvm-attachment",
                "-jar",
                initJar.getAbsolutePath(),
                "pid=" + this.vm.getPid(),
                "hostpid=" + this.vm.getPid(),
                "host=" + this.getAgentHost(),
                "port=" + agentHttpPort,
                "agentJar=" + agentJar.getAbsolutePath()
        };

        if (this.needsUserSwitch()) {
            attachCommand = this.addUserIdAndGroupId(attachCommand);
        }

        if (log.isDebugEnabled()) {
            log.debug("Executing attach command on host: {}", String.join(" ", attachCommand));
        }

        try {
            var process = this.startProcess(attachCommand);

            if (process.waitFor(ATTACH_PROCESS_TIMEOUT_SECONDS, TimeUnit.SECONDS)) {
                var rc = process.exitValue();
                if (rc != 0) {
                    var message = String.format("Attachment to JVM failed. ExitCode %d%nStdOut: %s%nStdErr: %s", rc, this.toString(process.getInputStream()),
                            this.toString(process.getErrorStream()));
                    throw new JvmAttachmentException(message);
                }
            } else {
                var message = String.format("Attachment to JVM timed out after %ds.", ATTACH_PROCESS_TIMEOUT_SECONDS);
                throw new JvmAttachmentException(message);
            }

            return true;
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw new JvmAttachmentException("External Attach failed.", e);
        } catch (IOException e) {
            throw new JvmAttachmentException("External Attach failed.", e);
        }
    }

    protected Process startProcess(String[] attachCommand) throws IOException {
        return new ProcessBuilder(attachCommand).start();
    }

    private boolean needsUserSwitch() {
        var agentProcess = SystemInfo.getOperatingSystem().getProcess(AgentUtils.getAgentPid());
        return agentProcess != null && !(agentProcess.getUserID().equals(this.vm.getUserId()) && agentProcess.getGroupID().equals(this.vm.getGroupId()));
    }

    private String[] addUserIdAndGroupId(String[] attachCommand) {
        if (this.vm.getGroupId() != null && this.vm.getUserId() != null) {
            return ArrayUtils.addAll(attachCommand, "uid=" + this.vm.getUserId(), "gid=" + this.vm.getGroupId());
        }
        return attachCommand;
    }

    private String toString(InputStream stream) {
        try {
            return IOUtils.toString(stream, Charset.defaultCharset());
        } catch (IOException e) {
            return e.getMessage();
        }
    }

    protected String getJavaExecutable() {
        if (StringUtils.isNotEmpty(this.vm.getPath()) && Files.isExecutable(Paths.get(this.vm.getPath()))) {
            return this.vm.getPath();
        }
        return "java";
    }

    @Override
    public void copyFiles(String dstPath, Map<String, File> files) {
        throw new UnsupportedOperationException();
    }

    @Override
    public String getAgentHost() {
        return "127.0.0.1";
    }
}
