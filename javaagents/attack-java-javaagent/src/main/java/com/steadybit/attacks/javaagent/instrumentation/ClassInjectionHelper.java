/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;


import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;
import com.steadybit.shaded.net.bytebuddy.dynamic.ClassFileLocator;
import com.steadybit.shaded.net.bytebuddy.dynamic.loading.ClassInjector;

import java.io.IOException;
import java.lang.instrument.Instrumentation;
import java.lang.reflect.InvocationTargetException;
import java.lang.reflect.Method;
import java.util.Collections;
import java.util.Map;
import java.util.Set;

public final class ClassInjectionHelper {
    private static final Logger log = RemoteAgentLogger.getLogger(ClassInjectionHelper.class);

    private ClassInjectionHelper() {
        //util
    }

    public static void inject(Instrumentation instrumentation, ClassLoader classLoader, Map<String, byte[]> classes) {
        if (ClassInjector.UsingUnsafe.isAvailable()) {
            new ClassInjector.UsingUnsafe(classLoader).injectRaw(classes);
        } else {
            injectUsingOpenedJavaLang(instrumentation, classLoader, classes);
        }
    }

    /**
     * Defines the classes through {@code ClassLoader#defineClass}. sun.misc.Unsafe, which ByteBuddy needs to get past
     * the encapsulation of java.lang, is disabled as of Java 26. An agent may open the package to itself instead.
     */
    static void injectUsingOpenedJavaLang(Instrumentation instrumentation, ClassLoader classLoader, Map<String, byte[]> classes) {
        try {
            openJavaLang(instrumentation);
            Method defineClass = ClassLoader.class.getDeclaredMethod("defineClass", String.class, byte[].class, int.class, int.class);
            defineClass.setAccessible(true);
            for (Map.Entry<String, byte[]> entry : classes.entrySet()) {
                try {
                    defineClass.invoke(classLoader, entry.getKey(), entry.getValue(), 0, entry.getValue().length);
                } catch (InvocationTargetException e) {
                    // A class loader rejects a second definition of a name, e.g. when two threads inject concurrently.
                    if (!(e.getCause() instanceof LinkageError) || !hasClass(classLoader, entry.getKey())) {
                        throw new IllegalStateException("Could not inject " + entry.getKey() + " into " + classLoader, e.getCause());
                    }
                }
            }
        } catch (ReflectiveOperationException e) {
            throw new IllegalStateException("Could not inject " + classes.keySet() + " into " + classLoader, e);
        }
    }

    // Uses reflection for the module API, as the agent is compiled for Java 8.
    private static void openJavaLang(Instrumentation instrumentation) throws ReflectiveOperationException {
        Method getModule = Class.class.getMethod("getModule");
        Object javaBase = getModule.invoke(ClassLoader.class);
        Object agentModule = getModule.invoke(ClassInjectionHelper.class);
        Instrumentation.class.getMethod("redefineModule", Class.forName("java.lang.Module"), Set.class, Map.class, Map.class, Set.class, Map.class)
                .invoke(instrumentation, javaBase, Collections.emptySet(), Collections.emptyMap(),
                        Collections.singletonMap("java.lang", Collections.singleton(agentModule)), Collections.emptySet(), Collections.emptyMap());
    }

    public static byte[] readClass(ClassLoader classLoader, String className) {
        try {
            ClassFileLocator.Resolution resolution = ClassFileLocator.ForClassLoader.of(classLoader).locate(className);
            if (!resolution.isResolved()) {
                throw new IllegalStateException("Class to inject not found: " + className);
            }
            return resolution.resolve();
        } catch (IOException e) {
            throw new IllegalStateException("Could not read class to inject: " + className, e);
        }
    }

    public static boolean hasClass(ClassLoader classLoader, String className) {
        try {
            classLoader.loadClass(className);
            return true;
        } catch (ClassNotFoundException e) {
            // Normal "class absent" case (e.g. probing for a Spring 6 marker on a Spring 5 target).
            log.trace("Class " + className + " not present in " + classLoader, e);
            return false;
        } catch (LinkageError e) {
            // Not the normal absent case: a linkage failure (conflicting or partially-shaded jars, incompatible
            // bytecode) points at a real problem, so log it instead of swallowing it. Still return false rather
            // than let the error escape and abort loading of the target's own class, since this runs inside
            // class matching/transformation.
            log.warn("Could not resolve " + className + " in " + classLoader + "; treating as absent", e);
            return false;
        }
    }
}
