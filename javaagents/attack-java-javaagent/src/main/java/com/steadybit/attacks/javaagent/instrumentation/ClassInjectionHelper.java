/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;


import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;
import com.steadybit.shaded.net.bytebuddy.dynamic.ClassFileLocator;
import com.steadybit.shaded.net.bytebuddy.dynamic.loading.ClassInjector;

import java.io.IOException;
import java.util.Map;

public final class ClassInjectionHelper {
    private static final Logger log = RemoteAgentLogger.getLogger(ClassInjectionHelper.class);

    private ClassInjectionHelper() {
        //util
    }

    public static void inject(ClassLoader classLoader, Map<String, byte[]> classes) {
        ClassInjector injector = ClassInjector.UsingUnsafe.isAvailable() ?
                new ClassInjector.UsingUnsafe(classLoader) :
                new ClassInjector.UsingReflection(classLoader);
        injector.injectRaw(classes);
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
