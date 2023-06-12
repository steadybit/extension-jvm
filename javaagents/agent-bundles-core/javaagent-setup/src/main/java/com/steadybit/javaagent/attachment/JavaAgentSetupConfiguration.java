/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment;

import com.steadybit.agent.resources.EmbeddedResourceHelper;
import com.steadybit.cri.CriClient;
import com.steadybit.docker.DockerClient;
import com.steadybit.javaagent.attachment.internal.DefaultJavaAgentFacade;
import com.steadybit.javaagent.attachment.internal.DefaultJavaVms;
import com.steadybit.javaagent.attachment.internal.HotspotJvmWatcher;
import com.steadybit.javaagent.attachment.internal.JavaProcessWatcher;
import com.steadybit.javaagent.attachment.internal.JvmAttachmentFactory;
import com.steadybit.javaagent.attachment.internal.RegisterJavaAgentHandler;
import com.steadybit.javaagent.attachment.internal.RemoteJvmConnections;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.boot.autoconfigure.AutoConfiguration;
import org.springframework.context.annotation.Bean;

@AutoConfiguration
public class JavaAgentSetupConfiguration {
    @Bean
    public RemoteJvmConnections remoteJvmConnections() {
        return new RemoteJvmConnections();
    }

    @Bean
    public JvmAttachmentFactory jvmAttachmentFactory(DockerClient dockerClient, CriClient criClient, JavaProcessWatcher javaProcessWatcher) {
        return new JvmAttachmentFactory(dockerClient, criClient, javaProcessWatcher);
    }

    @Bean
    public JavaAgentFacade javaAgentFacade(JavaProcessWatcher javaProcessWatcher, RemoteJvmConnections remoteJvmConnections,
            JvmAttachmentFactory jvmAttachmentFactory, JavaVms javaVms, @Value("${server.port}") int port,
            EmbeddedResourceHelper embeddedResourceHelper) {
        return new DefaultJavaAgentFacade(javaProcessWatcher, remoteJvmConnections, jvmAttachmentFactory, javaVms, embeddedResourceHelper, () -> port);
    }

    @Bean
    public JavaProcessWatcher javaProcessWatcher() {
        return new JavaProcessWatcher();
    }

    @Bean
    public RegisterJavaAgentHandler registerJavaAgentHandler(RemoteJvmConnections remoteJvmConnections) {
        return new RegisterJavaAgentHandler(remoteJvmConnections);
    }

    @Bean
    public HotspotJvmWatcher hotspotJvmWatcher() {
        return new HotspotJvmWatcher();
    }

    @Bean
    public JavaVms javaVms(HotspotJvmWatcher hotspotJvmWatcher, JavaProcessWatcher javaProcessWatcher, DockerClient dockerClient, CriClient criClient) {
        return new DefaultJavaVms(hotspotJvmWatcher, javaProcessWatcher, dockerClient, criClient);
    }
}
