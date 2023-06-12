/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal.jvmattach;

import com.steadybit.agent.commons.ProcFs;
import com.steadybit.containers.Container;
import com.steadybit.containers.ExecResult;
import com.steadybit.javaagent.attachment.JavaVm;
import com.steadybit.javaagent.attachment.JvmAttachmentException;
import com.steadybit.utils.ContainerNetworkUtil;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.File;
import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.StandardCopyOption;
import java.util.HashMap;
import java.util.Map;
import java.util.Optional;

public abstract class ContainerJvmAttachment implements JvmAttachment {
    protected final Logger log = LoggerFactory.getLogger(this.getClass());
    protected final JavaVm vm;

    protected ContainerJvmAttachment(JavaVm vm) {
        this.vm = vm;
    }

    @Override
    public boolean attach(File agentJar, File initJar, int agentHttpPort) {
        var container = this.getContainer();
        if (!container.isPresent()) {
            this.log.debug("Container not present. Skipping attachment to JVM {}", this.vm);
            return false;
        }

        if (!container.get().isRunning()) {
            this.log.debug("Container not running. Skipping attachment to JVM {}", this.vm);
            return false;
        }

        var labels = container.get().getLabels();
        if ("false".equals(labels.get("com.steadybit.agent/jvm-attach"))) {
            this.log.info("Container label suppressing attachment. Skipping attachment to JVM {}", this.vm);
            return false;
        }

        if ("true".equals(labels.get("com.steadybit.sidecar"))) {
            this.log.debug("Skipping attachment to JVM in steadybit-sidecar container {} ", this.vm);
            return false;
        }

        this.log.debug("Attaching to JVM: {}", this.vm);
        Map<String, File> files = new HashMap<>();
        files.put("steadybit-javaagent-main.jar", agentJar);
        files.put("steadybit-javaagent-init.jar", initJar);
        this.copyFiles("/tmp", files);

        var attachCommand = new String[] {
                this.getJavaExecutable(),
                "-Xms16m",
                "-Xms16m",
                "-XX:+UseSerialGC",
                "-XX:+PerfDisableSharedMem",
                "-Dsun.tools.attach.attachTimeout=30000",
                "-Dsteadybit.agent.disable-jvm-attachment",
                "-jar",
                "/tmp/steadybit-javaagent-init.jar",
                "pid=" + this.vm.getInContainerPid(),
                "hostpid=" + this.vm.getPid(),
                "host=" + this.getAgentHost(),
                "port=" + agentHttpPort,
                "agentJar=/tmp/steadybit-javaagent-main.jar"
        };

        if (this.log.isDebugEnabled()) {
            this.log.debug("Executing attach command in container {}: {}", this.vm.getContainerId(), String.join(" ", attachCommand));
        }
        var execResult = this.exec(attachCommand);
        if (execResult.getExitCode() != 0) {
            var message = String.format("Attachment to JVM failed. ExitCode %d%nStdOut: %s%nStdErr: %s", execResult.getExitCode(), execResult.getStdOut(),
                    execResult.getStdErr());
            throw new JvmAttachmentException(message);
        }
        return true;
    }

    @Override
    public void copyFiles(String dstPath, Map<String, File> files) {
        var destDir = ProcFs.ROOT.getProcessRoot(this.vm.getPid()).resolve((dstPath.startsWith("/") ? "." : "") + dstPath);
        try {
            for (var entry : files.entrySet()) {
                var destFile = destDir.resolve(entry.getKey()).toFile();
                var srcFile = entry.getValue();
                if (destFile.lastModified() == srcFile.lastModified()) {
                    continue;
                }

                Files.copy(srcFile.toPath(), destFile.toPath(), StandardCopyOption.REPLACE_EXISTING);
                this.log.debug("Copied {} to container {} {}", srcFile, this.vm.getContainerId(), destFile);
                if (!destFile.setLastModified(srcFile.lastModified())) {
                    this.log.warn("Failed to set {} last modified", destFile);
                }
                if (!destFile.setReadable(true, false)) {
                    this.log.debug("Failed to make {} readable", destFile);
                }
                if (!destFile.setExecutable(true, false)) {
                    this.log.debug("Failed to make {} executable", destFile);
                }
                if (!destFile.setWritable(true, false)) {
                    this.log.debug("Failed to make {} writable", destFile);
                }
            }
        } catch (IOException e) {
            throw new JvmAttachmentException("Could not copy files to container", e);
        }
    }

    @Override
    public String getAgentHost() {
        return ContainerNetworkUtil.getAgentPublicAddress();
    }

    protected String getJavaExecutable() {
        if (this.vm.getPath() != null) {
            try {
                var result = this.exec(new String[] { this.vm.getPath(), "-version" });
                if (result.getExitCode() == 0) {
                    this.log.trace("Using java executable {} in container", this.vm.getPath());
                    return this.vm.getPath();
                }

            } catch (Exception ex) {
                this.log.debug("Cannot use java executable {} in container", this.vm.getPath(), ex);
            }
        }
        return "java";
    }

    protected abstract ExecResult exec(String[] command);

    protected abstract Optional<Container> getContainer();
}
