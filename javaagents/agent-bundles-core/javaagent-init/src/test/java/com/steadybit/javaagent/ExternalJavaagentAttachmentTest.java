/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent;

import net.bytebuddy.agent.ByteBuddyAgent;
import static org.assertj.core.api.Assertions.assertThatCode;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.mockito.MockedStatic;
import static org.mockito.Mockito.mockStatic;

import java.io.File;
import java.net.URISyntaxException;
import java.nio.file.Path;

class ExternalJavaagentAttachmentTest {
    @TempDir
    Path tempDir;

    @Test
    void should_attach_with_hostpid() throws URISyntaxException {
        File ownJar = new File(ExternalJavaagentAttachment.class.getProtectionDomain().getCodeSource().getLocation().toURI());

        try (MockedStatic<ByteBuddyAgent> agent = mockStatic(ByteBuddyAgent.class)) {
            assertThatCode(() -> {
                String[] args = { "pid=1", "hostpid=1234", "host=127.0.0.1", "port=42899", "agentJar=" + this.tempDir };
                ExternalJavaagentAttachment.main(args);
            }).doesNotThrowAnyException();

            agent.verify(() -> ByteBuddyAgent.attach(ownJar, "1", "agentJar=" + this.tempDir + ",pid=1234,host=127.0.0.1,port=42899"));
        }
    }

    @Test
    void should_attach_without_hostpid() throws URISyntaxException {
        File ownJar = new File(ExternalJavaagentAttachment.class.getProtectionDomain().getCodeSource().getLocation().toURI());

        try (MockedStatic<ByteBuddyAgent> agent = mockStatic(ByteBuddyAgent.class)) {
            assertThatCode(() -> {
                String[] args = { "pid=1", "host=127.0.0.1", "port=42899", "agentJar=" + this.tempDir };
                ExternalJavaagentAttachment.main(args);
            }).doesNotThrowAnyException();

            agent.verify(() -> ByteBuddyAgent.attach(ownJar, "1", "agentJar=" + this.tempDir + ",pid=1,host=127.0.0.1,port=42899"));
        }
    }
}