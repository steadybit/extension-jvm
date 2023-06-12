/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import static org.assertj.core.api.Assertions.assertThat;
import static org.awaitility.Awaitility.await;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;
import org.junitpioneer.jupiter.RetryingTest;
import org.springframework.core.io.FileSystemResource;
import org.testcontainers.junit.jupiter.Testcontainers;

@Testcontainers
class DefaultJavaAgentFacadeHostITest {
    private static final SpringBootSampleProcess hostProcessPerfData = new SpringBootSampleProcess(true);
    private static final SpringBootSampleProcess hostProcessNoPerfData = new SpringBootSampleProcess(false);
    private static final StandaloneJavaAgentFacade javaAgentFacade = new StandaloneJavaAgentFacade();

    @BeforeAll
    static void setup() {
        javaAgentFacade.addResourceOverride(DefaultJavaAgentFacade.JAVAAGENT_INIT_JAR, new FileSystemResource("target/javaagent/javaagent-init.jar"));
        javaAgentFacade.addResourceOverride(DefaultJavaAgentFacade.JAVAAGENT_MAIN_JAR, new FileSystemResource("target/javaagent/javaagent-main.jar"));
        javaAgentFacade.start();
        hostProcessPerfData.start();
        hostProcessNoPerfData.start();
    }

    @AfterAll
    static void destroy() {
        javaAgentFacade.stop();
        hostProcessNoPerfData.stop();
        hostProcessPerfData.stop();
    }

    @Test
    @RetryingTest(2)
    void attach_to_host_process_with_perfdata() {
        var jvm = hostProcessPerfData.getJavaVm();

        javaAgentFacade.waitForAttachment(jvm);

        await().untilAsserted(() -> {
            assertThat(javaAgentFacade.isAttached(jvm)).isTrue();
            assertThat(javaAgentFacade.hasClassLoaded(jvm, "org.springframework.boot.ApplicationRunner")).isTrue();
        });
    }

    @Test
    @RetryingTest(2)
    void attach_to_host_process_without_perfdata() {
        var jvm = hostProcessNoPerfData.getJavaVm();
        javaAgentFacade.waitForAttachment(jvm);

        await().untilAsserted(() -> {
            assertThat(javaAgentFacade.isAttached(jvm)).isTrue();
            assertThat(javaAgentFacade.hasClassLoaded(jvm, "org.springframework.boot.ApplicationRunner")).isTrue();
        });
    }
}