/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.spring.instrumentation;


import com.steadybit.shaded.net.bytebuddy.dynamic.ClassFileLocator;

import java.io.IOException;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public final class ClassInjectionHelper {
    private ClassInjectionHelper() {
        //util
    }

    public static Map<String, byte[]> resolveClasses(List<String> names, ClassLoader classLoader) {
        Map<String, byte[]> resolvedClasses = new HashMap<>();
        ClassFileLocator locator = ClassFileLocator.ForClassLoader.of(classLoader);
        try {
            for (String name : names) {
                ClassFileLocator.Resolution resolution = locator.locate(name);
                if (resolution.isResolved()) {
                    resolvedClasses.put(name, resolution.resolve());
                }
            }
        } catch (IOException e) {
            throw new IllegalStateException("Could not locate classes to inject", e);
        }
        return resolvedClasses;
    }

    public static boolean hasClass(ClassLoader classLoader, String className) {
        try {
            classLoader.loadClass(className);
            return true;
        } catch (ClassNotFoundException e) {
            return false;
        }
    }
}
