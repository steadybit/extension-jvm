/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent;

import org.apache.commons.io.IOUtils;
import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;
import org.junit.jupiter.api.Test;

import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.lang.reflect.InvocationTargetException;
import java.util.ArrayList;
import java.util.List;
import java.util.jar.Attributes;
import java.util.jar.JarEntry;
import java.util.jar.JarOutputStream;
import java.util.jar.Manifest;

class TrampolineAgentTest {
    public static List<String> events = new ArrayList<>();

    @Test
    void should_load_agent() throws Exception {
        File agentV1 = this.createAgentJar(TestAgentV1.class);
        File agentV2 = this.createAgentJar(TestAgentV2.class);

        TrampolineAgent.agentmain("agentJar=" + agentV1 + ",additionalParms", null);
        TrampolineAgent.agentmain("agentJar=" + agentV2 + ",additionalParms", null);

        assertThat(events).contains("Loaded TestAgentV1", "Stopped TestAgentV1", "Loaded TestAgentV2");
    }

    @Test
    void should_propagate_exception() throws Exception {
        File agent = this.createAgentJar(TestAgentThrowing.class);

        assertThatThrownBy(() -> TrampolineAgent.agentmain("agentJar=" + agent + ",additionalParms", null))
                .isInstanceOf(InvocationTargetException.class).hasCauseInstanceOf(RuntimeException.class);
    }

    private File createAgentJar(Class<?> agentClass) throws IOException {
        File file = File.createTempFile("test-agent", ".jar");
        Manifest manifest = new Manifest();

        Attributes attributes = manifest.getMainAttributes();
        attributes.put(Attributes.Name.MANIFEST_VERSION, "1.0.0");
        attributes.putValue("Agent-Class", agentClass.getName());
        try (JarOutputStream jar = new JarOutputStream(new FileOutputStream(file), manifest)) {
            String classLocation = agentClass.getName().replace(".", "/") + ".class";
            jar.putNextEntry(new JarEntry(classLocation));
            try (InputStream classIs = agentClass.getResourceAsStream("/" + classLocation)) {
                IOUtils.copy(classIs, jar);
            }
            jar.closeEntry();
        }
        return file;
    }
}