/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.handler;

import com.steadybit.javaagent.AgentPlugin;
import com.steadybit.javaagent.CommandHandler;
import com.steadybit.javaagent.LoadedClassesCache;
import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;
import net.bytebuddy.dynamic.loading.MultipleParentClassLoader;

import java.io.Closeable;
import java.io.File;
import java.io.IOException;
import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.io.PrintWriter;
import java.lang.instrument.Instrumentation;
import java.net.MalformedURLException;
import java.net.URL;
import java.net.URLClassLoader;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.concurrent.ConcurrentHashMap;
import java.util.jar.JarFile;

public class LoadAgentPluginCommandHandler implements CommandHandler {
    private static final Logger log = RemoteAgentLogger.getLogger(LoadAgentPluginCommandHandler.class);
    private final ConcurrentHashMap<String, AgentPluginInstance> loadedAgentPlugins = new ConcurrentHashMap<>();
    private final Instrumentation instrumentation;
    private final LoadedClassesCache loadedClassesCache;

    public LoadAgentPluginCommandHandler(Instrumentation instrumentation, LoadedClassesCache loadedClassesCache) {
        this.instrumentation = instrumentation;
        this.loadedClassesCache = loadedClassesCache;
    }

    @Override
    public boolean canHandle(String command) {
        if ("load-agent-plugin".equals(command) || "unload-agent-plugin".equals(command)) {
            return true;
        }

        for (AgentPluginInstance agentPluginInstance : this.loadedAgentPlugins.values()) {
            if (agentPluginInstance.canHandle(command)) {
                return true;
            }
        }

        return false;
    }

    @Override
    public void handle(String command, String argument, OutputStream os) {
        try {
            PrintWriter writer = new PrintWriter(new OutputStreamWriter(os, StandardCharsets.UTF_8), true);
            switch (command) {
            case "load-agent-plugin":
                boolean loadResult = this.loadAgentPlugin(argument);
                writer.write(RC_OK);
                writer.println(loadResult);
                break;
            case "unload-agent-plugin":
                boolean unloadResult = this.unloadAgentPlugin(argument);
                writer.write(RC_OK);
                writer.println(unloadResult);
                break;
            default:
                for (AgentPluginInstance agentPluginInstance : this.loadedAgentPlugins.values()) {
                    if (agentPluginInstance.canHandle(command)) {
                        agentPluginInstance.handle(command, argument, os);
                    }
                }
            }
        } catch (Exception e) {
            log.error("Could not execute command '" + command + "': ", e);
        }
    }

    public void destroy() {
        for (AgentPluginInstance agentPluginInstance : new ArrayList<>(this.loadedAgentPlugins.values())) {
            agentPluginInstance.destroy();
        }
    }

    private boolean loadAgentPlugin(String argument) {
        String[] tokens = argument.split(",", 2);
        File agentJar = new File(tokens[0]);
        String agentArgs = tokens.length > 1 ? tokens[1] : "";

        try {
            this.unloadAgentPlugin(agentJar.toString());
        } catch (Exception e) {
            log.warn("Failed to close old agent plugin " + agentJar);
        }

        AgentPluginInstance instance = this.invokeAgentPlugin(agentJar, agentArgs);
        this.loadedAgentPlugins.put(agentJar.toString(), instance);
        return true;
    }

    @SuppressWarnings("java:S2095")
    private AgentPluginInstance invokeAgentPlugin(File agentJar, String agentArgs) {
        String agentClass = this.getAgentPluginClass(agentJar);
        ClassLoader parentClassLoader = this.getAgentPluginParentClassLoader(agentJar);
        AgentURLClassloader classLoader = new AgentURLClassloader(agentJar, parentClassLoader);

        try {
            Class<?> clazz = classLoader.loadClass(agentClass);

            Object agent;
            try {
                agent = clazz.getConstructor(String.class, Instrumentation.class).newInstance(agentArgs, this.instrumentation);
            } catch (NoSuchMethodException ex1) {
                try {
                    agent = clazz.getConstructor(String.class).newInstance(agentArgs);
                } catch (NoSuchMethodException ex2) {
                    try {
                        agent = clazz.getConstructor(Instrumentation.class).newInstance(this.instrumentation);
                    } catch (NoSuchMethodException ex3) {
                        agent = clazz.getConstructor().newInstance();
                    }
                }
            }

            if (agent instanceof AgentPlugin) {
                ((AgentPlugin) agent).start();
            }

            return new AgentPluginInstance(agent, classLoader);
        } catch (Exception e) {
            try {
                classLoader.close();
            } catch (IOException ioException) {
                e.addSuppressed(ioException);
            }
            throw new RuntimeException("Could not instantiate and start agent plugin", e);
        }
    }

    private ClassLoader getAgentPluginParentClassLoader(File agentJar) {
        Class<?> classLoaderReference = this.getAgentParentClassLoaderReference(agentJar);
        MultipleParentClassLoader.Builder builder = new MultipleParentClassLoader.Builder();
        if (classLoaderReference != null) {
            builder = builder.append(classLoaderReference);
        }
        builder = builder.append(this.getClass());
        return builder.build();
    }

    private boolean unloadAgentPlugin(String argument) {
        String[] tokens = argument.split(",", 2);
        File agentJar = new File(tokens[0]);
        boolean deleteFile = tokens.length > 1 && "deleteFile=true".equals(tokens[1]);

        AgentPluginInstance agentPluginInstance = this.loadedAgentPlugins.remove(agentJar.toString());
        if (agentPluginInstance != null) {
            try {
                agentPluginInstance.destroy();
                return true;
            } finally {
                if (deleteFile) {
                    if (!agentJar.delete()) {
                        log.debug("Failed to delete agent plugin: {}", agentJar);
                    }
                }
            }
        }
        return false;
    }

    private String getAgentPluginClass(File agentJar) {
        try (JarFile jar = new JarFile(agentJar)) {
            return jar.getManifest().getMainAttributes().getValue("Agent-Plugin-Class");
        } catch (IOException e) {
            throw new RuntimeException("Could not determine Agent-Class", e);
        }
    }

    private Class<?> getAgentParentClassLoaderReference(File agentJar) {
        try (JarFile jar = new JarFile(agentJar)) {
            String className = jar.getManifest().getMainAttributes().getValue("Agent-ClassLoader-Of");
            if (className != null) {
                return this.loadedClassesCache.findClass(className);
            }
            return null;
        } catch (IOException e) {
            throw new RuntimeException("Could not determine Agent-Class-Loader-Of", e);
        }
    }
}

class AgentPluginInstance implements CommandHandler {
    private static final Logger log = RemoteAgentLogger.getLogger(LoadAgentPluginCommandHandler.class);

    private final Object instance;
    private final ClassLoader classLoader;

    AgentPluginInstance(Object instance, ClassLoader classLoader) {
        this.instance = instance;
        this.classLoader = classLoader;
    }

    public void destroy() {
        try {
            if (this.classLoader instanceof Closeable) {
                ((Closeable) this.classLoader).close();
            }
            if (this.instance instanceof AgentPlugin) {
                ((AgentPlugin) this.instance).destroy();
            }
        } catch (Exception e) {
            log.warn("Error while closing AgentPlugin " + this.instance.getClass().getName());
        }
    }

    @Override
    public boolean canHandle(String command) {
        if (this.instance instanceof CommandHandler) {
            return ((CommandHandler) this.instance).canHandle(command);
        }
        return false;
    }

    @Override
    public void handle(String command, String argument, OutputStream os) {
        if (this.instance instanceof CommandHandler) {
            ((CommandHandler) this.instance).handle(command, argument, os);
        }
    }
}

class AgentURLClassloader extends URLClassLoader {
    static {
        ClassLoader.registerAsParallelCapable();
    }

    public AgentURLClassloader(File agentJar, ClassLoader parent) {
        super(new URL[] { toURL(agentJar) }, parent);
    }

    private static URL toURL(File agentJar) {
        try {
            return agentJar.toURI().toURL();
        } catch (MalformedURLException e) {
            throw new RuntimeException("Could not load agent JAR because it did not translate to an URL");
        }
    }
}