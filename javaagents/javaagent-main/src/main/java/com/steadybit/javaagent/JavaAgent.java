/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent;

import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;
import com.steadybit.javaagent.util.TempFileUtils;
import net.bytebuddy.dynamic.ClassFileLocator;
import net.bytebuddy.dynamic.loading.ClassInjector;

import java.io.File;
import java.io.IOException;
import java.lang.instrument.Instrumentation;
import java.lang.reflect.InvocationTargetException;
import java.net.URL;
import java.util.Arrays;
import java.util.HashMap;
import java.util.Map;
import java.util.Set;
import java.util.stream.Collectors;

public class JavaAgent {
    private static final Logger log = RemoteAgentLogger.getLogger(JavaAgent.class);
    private static JavaAgentSocket javaAgentSocket;

    /**
     * Stops the previous agent (if supplied) and starts the main thread of this agent.
     */
    public static void init(String agentArguments, Instrumentation instrumentation, ClassLoader previousAgent) throws Exception {
        String inject = getValueFromArgument(agentArguments, "disableBootstrapLoaderInjection");
        if ("true".equals(inject)) {
            log.info("Injection of Steadybit classes into bootstrap loader disabled.");
        } else {
            injectClassesIntoBootstrapLoader(instrumentation);
        }

        String pid = getValueFromArgument(agentArguments, "pid");
        String host = getValueFromArgument(agentArguments, "host");
        String port = getValueFromArgument(agentArguments, "port");
        URL url = new URL(String.format("http://%s:%s/javaagent", host, port));
        RemoteAgentLogger.init(pid, new URL(url + "/log"));

        String heartbeat = getValueFromArgument(agentArguments, "heartbeat");
        File heartbeatFile = heartbeat != null ? new File(heartbeat) : null;

        boolean previousAgentStopped = stopPreviousAgent(previousAgent);
        if (previousAgentStopped) {
            startThread(pid, url, heartbeatFile, instrumentation);
        }
    }

    private static void startThread(String pid, URL url, File heartbeat, Instrumentation instrumentation) {
        javaAgentSocket = new JavaAgentSocket(pid, url, heartbeat, new JavaAgentSocketHandler(instrumentation));
        javaAgentSocket.start();
        log.debug(String.format("JavaAgent started for PID %s.", pid));
    }

    private static boolean stopPreviousAgent(ClassLoader previousAgent) throws ClassNotFoundException {
        if (previousAgent == null) {
            return true;
        }
        try {
            log.debug("Stopping previous attached agent");
            previousAgent.loadClass("com.steadybit.javaagent.JavaAgent").getMethod("stop").invoke(null);
            return true;
        } catch (NoSuchMethodException | InvocationTargetException | IllegalAccessException e) {
            log.error("Failed to remove old agent");
            return false;
        }
    }

    public static void stop() {
        if (javaAgentSocket == null || javaAgentSocket.isAlive()) {
            return;
        }

        log.debug("Stopping Agent thread");

        try {
            javaAgentSocket.shutdown(true);
        } catch (Throwable ex) {
            log.error("Failed shutdown JavaAgent", ex);
        }
    }

    private static String getValueFromArgument(String agentArguments, String key) {
        if (agentArguments != null) {
            for (String param : agentArguments.split(",")) {
                String[] tokens = param.split("=", 2);
                if (key.equals(tokens[0]) && tokens.length == 2) {
                    return tokens[1];
                }
            }
        }
        return null;
    }

    private static void injectClassesIntoBootstrapLoader(Instrumentation instrumentation) throws IOException {
        Map<String, byte[]> classes = locateAndFilterClasses(instrumentation, "com.steadybit.javaagent.instrumentation.InstrumentationPluginDispatcher",
                "com.steadybit.javaagent.instrumentation.InstrumentationPlugin");
        if (classes.isEmpty()) {
            log.warn("No Steadybit classes eligible for injecting");
        }

        ClassInjector injector = createClassInjector(instrumentation);
        if (injector != null) {
            log.debug("Injecting Steadybit classes using %s injector", injector.getClass());
            injector.injectRaw(classes);
        } else {
            log.warn("No injector available to inject Steadybit classes");
        }
    }

    private static Map<String, byte[]> locateAndFilterClasses(Instrumentation instrumentation, String... classNames) throws IOException {
        try (ClassFileLocator classFileLocator = ClassFileLocator.ForClassLoader.of(JavaAgent.class.getClassLoader())) {
            Set<String> knownBootstrapClasses = Arrays.stream(instrumentation.getInitiatedClasses(null)).map(Class::getName).collect(Collectors.toSet());

            Map<String, byte[]> classes = new HashMap<>();
            for (String className : classNames) {
                if (!knownBootstrapClasses.contains(className)) {
                    try {
                        ClassFileLocator.Resolution resolution = classFileLocator.locate(className);
                        if (resolution.isResolved()) {
                            classes.put(className, resolution.resolve());
                        } else {
                            log.warn("Could not locate class %s for injection", className);
                        }
                    } catch (IOException e) {
                        log.error("Could not locate class " + className + " for injection", e);
                    }
                }
            }
            return classes;
        }
    }

    private static ClassInjector createClassInjector(Instrumentation instrumentation) {
        if (ClassInjector.UsingUnsafe.isAvailable()) {
            return ClassInjector.UsingUnsafe.ofBootLoader();
        } else if (ClassInjector.UsingInstrumentation.isAvailable()) {
            return ClassInjector.UsingInstrumentation.of(getTempDir(), ClassInjector.UsingInstrumentation.Target.BOOTSTRAP, instrumentation);
        } else {
            return null;
        }
    }

    private static File getTempDir() {
        try {
            return TempFileUtils.createTempDir("sb-agent");
        } catch (IOException e) {
            throw new RuntimeException(e);
        }
    }
}
