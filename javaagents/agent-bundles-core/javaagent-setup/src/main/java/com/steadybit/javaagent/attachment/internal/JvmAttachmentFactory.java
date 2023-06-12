/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import com.steadybit.cri.CriClient;
import com.steadybit.docker.DockerClient;
import com.steadybit.javaagent.attachment.JavaVm;
import com.steadybit.javaagent.attachment.internal.jvmattach.CriContainerJvmAttachment;
import com.steadybit.javaagent.attachment.internal.jvmattach.DockerContainerJvmAttachment;
import com.steadybit.javaagent.attachment.internal.jvmattach.HostVmAttachment;
import com.steadybit.javaagent.attachment.internal.jvmattach.JvmAttachment;

public class JvmAttachmentFactory {
    private final DockerClient dockerClient;
    private final CriClient criClient;
    private final JavaProcessWatcher javaProcessWatcher;

    public JvmAttachmentFactory(DockerClient dockerClient, CriClient criClient,
            JavaProcessWatcher javaProcessWatcher) {
        this.dockerClient = dockerClient;
        this.criClient = criClient;
        this.javaProcessWatcher = javaProcessWatcher;
    }

    public JvmAttachment getAttachment(JavaVm jvm) {
        if (!jvm.isRunningInContainer()) {
            return new HostVmAttachment(jvm, this.javaProcessWatcher);
        }

        if (DockerClient.PREFIX.matches(jvm.getContainerId())) {
            return new DockerContainerJvmAttachment(jvm, this.dockerClient);
        }

        if (CriClient.PREFIX.matches(jvm.getContainerId())) {
            return new CriContainerJvmAttachment(jvm, this.criClient);
        }

        throw new IllegalArgumentException("Container Runtime not supported for " + jvm.getContainerId());
    }
}
