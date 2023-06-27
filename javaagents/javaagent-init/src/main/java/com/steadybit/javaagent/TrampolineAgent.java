/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent;

import java.io.File;
import java.io.IOException;
import java.lang.instrument.Instrumentation;
import java.net.MalformedURLException;
import java.net.URL;
import java.net.URLClassLoader;
import java.util.jar.JarFile;

/**
 * This is a "trampoline" agent with loads the actual agent class in a new classloader that can be disposed.
 */
public class TrampolineAgent {
    static URLClassLoader previous;

    @SuppressWarnings({ "java:S2095", "java:S2093" }) //Closing of URLClassLoader handled with love
    public static void agentmain(String agentArguments, Instrumentation instrumentation) throws Exception {
        URLClassLoader classLoader = new URLClassLoader(new URL[] { getAgentJarURL(agentArguments) });
        try {
            classLoader.loadClass(getAgentClass(agentArguments))
                    .getMethod("init", String.class, Instrumentation.class, ClassLoader.class)
                    .invoke(null, agentArguments, instrumentation, previous);
        } catch (Exception e) {
            classLoader.close();
            throw e;
        } finally {
            if (previous != null) {
                previous.close();
            }
            previous = classLoader;
        }
    }

    private static URL getAgentJarURL(String agentArguments) throws MalformedURLException {
        String agentJar = getValueFromArgument(agentArguments, "agentJar");
        if (agentJar != null) {
            return new URL("file:" + agentJar);
        }
        throw new IllegalArgumentException("Argument agentJar must not be empty!");
    }

    private static String getAgentClass(String agentArguments) {
        String agentJar = getValueFromArgument(agentArguments, "agentJar");

        try (JarFile jar = new JarFile(new File(agentJar))) {
            return jar.getManifest().getMainAttributes().getValue("Agent-Class");
        } catch (IOException e) {
            throw new IllegalArgumentException("Could not determine Agent-Class in " + agentJar, e);
        }
    }

    private static String getValueFromArgument(String agentArguments, String key) {
        if (agentArguments != null) {
            for (String param : agentArguments.split(",")) {
                String[] keyValue = param.split("=", 2);
                if (key.equals(keyValue[0]) && keyValue.length == 2) {
                    return keyValue[1];
                }
            }
        }
        return null;
    }
}