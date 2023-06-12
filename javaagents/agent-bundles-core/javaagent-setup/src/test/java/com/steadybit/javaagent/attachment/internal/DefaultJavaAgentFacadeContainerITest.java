/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import static org.assertj.core.api.Assertions.assertThat;
import static org.awaitility.Awaitility.await;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;
import org.springframework.core.io.FileSystemResource;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;

import java.net.InetSocketAddress;

@Testcontainers
class DefaultJavaAgentFacadeContainerITest {
    @Container
    private static final SpringBootSampleContainer container = new SpringBootSampleContainer();
    @Container
    private static final SpringBootSampleContainer containerNoAttach = new SpringBootSampleContainer().withLabel("com.steadybit.agent/jvm-attach", "false");
    @Container
    private static final SpringBootSampleContainer containerSidecar = new SpringBootSampleContainer().withLabel("com.steadybit.sidecar", "true");
    private static final StandaloneJavaAgentFacade javaAgentFacade = new StandaloneJavaAgentFacade();

    @BeforeAll
    static void setup() {
        javaAgentFacade.addResourceOverride(DefaultJavaAgentFacade.JAVAAGENT_INIT_JAR, new FileSystemResource("target/javaagent/javaagent-init.jar"));
        javaAgentFacade.addResourceOverride(DefaultJavaAgentFacade.JAVAAGENT_MAIN_JAR, new FileSystemResource("target/javaagent/javaagent-main.jar"));
        javaAgentFacade.start();
    }

    @AfterAll
    static void destroy() {
        javaAgentFacade.stop();
    }

    @Test
    void should_attach_to_container() {
        var jvm = container.getJavaVm();

        javaAgentFacade.setProxyRemoteJvmConnection((pid, connection) -> {
            if (pid == jvm.getPid() && System.getProperty("os.name").startsWith("Mac OS X")) {
                return this.startTcpProxy(container.getContainerInfo().getNetworkSettings().getIpAddress(), connection.getPort());
            }
            return connection;
        });

        javaAgentFacade.addJvm(jvm);
        javaAgentFacade.waitForAttachment(jvm);

        await().untilAsserted(() -> {
            assertThat(javaAgentFacade.isAttached(container.getJavaVm())).isTrue();
            assertThat(javaAgentFacade.hasClassLoaded(jvm, "org.springframework.boot.ApplicationRunner")).isTrue();
        });
    }

    @Test
    void should_not_attach_to_container_suppressed_by_label() {
        assertThat(javaAgentFacade.attachInternal(containerNoAttach.getJavaVm())).isFalse();
        assertThat(javaAgentFacade.isAttached(containerNoAttach.getJavaVm())).isFalse();
    }

    @Test
    void should_not_attach_to_sidecar_container() {
        assertThat(javaAgentFacade.attachInternal(containerSidecar.getJavaVm())).isFalse();
        assertThat(javaAgentFacade.isAttached(containerSidecar.getJavaVm())).isFalse();
    }

    private InetSocketAddress startTcpProxy(String host, int port) {
        var proxy = new GenericContainer<>("alpine/socat")
                .withExposedPorts(9090)
                .withCommand(String.format("tcp-listen:9090,fork,reuseaddr tcp-connect:%s:%s", host, port));
        proxy.start();
        return new InetSocketAddress("localhost", proxy.getMappedPort(9090));
    }
}