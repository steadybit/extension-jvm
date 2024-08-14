/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.handler;

import com.steadybit.javaagent.CommandHandler;
import com.steadybit.javaagent.log.RemoteAgentLogger;
import org.apache.commons.io.IOUtils;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.io.BufferedReader;
import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.FileOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.util.jar.Attributes;
import java.util.jar.JarEntry;
import java.util.jar.JarOutputStream;
import java.util.jar.Manifest;

import static org.assertj.core.api.Assertions.assertThat;

class LoadAgentPluginCommandHandlerTest {
    private final CommandHandler handler = new LoadAgentPluginCommandHandler(null, null);
    public static boolean pluginLoaded = false;
    public static boolean pluginUnloaded = false;

    @BeforeEach
    void setUp() {
        RemoteAgentLogger.setLogToSystem(true);
    }

    @Test
    void should_load_and_unload_plugin() throws Exception {
        //when
        File pluginLocation = this.createPluginJar();
        String result = this.command("load-agent-plugin", pluginLocation.toString());
        //then
        assertThat(result).isEqualTo("true");
        assertThat(pluginLoaded).isTrue();

        //when
        result = this.command("unload-agent-plugin", pluginLocation.toString());
        assertThat(pluginUnloaded).isTrue();
        //then
        assertThat(result).isEqualTo("true");
    }

    private File createPluginJar() throws IOException {
        File file = File.createTempFile("test-agent", ".jar");

        String classLocation = TestAgentPlugin.class.getName().replace(".", "/") + ".class";
        Manifest manifest = new Manifest();
        Attributes attributes = manifest.getMainAttributes();
        attributes.putValue("Agent-Plugin-Class", TestAgentPlugin.class.getName());
        attributes.put(Attributes.Name.MANIFEST_VERSION, "1.0.0");
        try (JarOutputStream jar = new JarOutputStream(new FileOutputStream(file), manifest)) {
            jar.putNextEntry(new JarEntry(classLocation));
            try (InputStream classIs = TestAgentPlugin.class.getResourceAsStream("/" + classLocation)) {
                IOUtils.copy(classIs, jar);
            }
            jar.closeEntry();
        }
        return file;
    }

    private String command(String command, String argument) throws IOException {
        ByteArrayOutputStream os = new ByteArrayOutputStream();
        this.handler.handle(command, argument, os);
        byte[] buf = os.toByteArray();
        assertThat(buf[0]).isEqualTo(CommandHandler.RC_OK);
        return new BufferedReader(new InputStreamReader(new ByteArrayInputStream(buf, 1, buf.length - 1))).readLine();
    }
}