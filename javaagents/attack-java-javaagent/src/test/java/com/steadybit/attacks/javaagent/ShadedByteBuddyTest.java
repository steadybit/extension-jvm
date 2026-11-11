/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent;

import com.steadybit.shaded.net.bytebuddy.agent.builder.AgentBuilder;
import org.junit.jupiter.api.Test;

import java.io.File;
import java.util.List;
import java.util.jar.JarFile;
import java.util.stream.Collectors;

import static org.assertj.core.api.Assertions.assertThat;

/**
 * ByteBuddy reads class files through classes in the multi-release part of its jar when the JVM is newer than its
 * bundled ASM. If those are not relocated along with the rest, instrumentation fails on such JVMs.
 */
class ShadedByteBuddyTest {

    @Test
    void should_relocate_multi_release_classes() throws Exception {
        File shadedJar = new File(AgentBuilder.class.getProtectionDomain().getCodeSource().getLocation().toURI());

        try (JarFile jar = new JarFile(shadedJar)) {
            List<String> versionedClasses = jar.stream()
                    .map(entry -> entry.getName())
                    .filter(name -> name.startsWith("META-INF/versions/") && name.endsWith(".class"))
                    .collect(Collectors.toList());

            assertThat(versionedClasses).isNotEmpty()
                    .allMatch(name -> name.matches("META-INF/versions/\\d+/com/steadybit/shaded/net/bytebuddy/.*"));
            assertThat(jar.getManifest().getMainAttributes().getValue("Multi-Release")).isEqualTo("true");
        }
    }
}
