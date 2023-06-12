/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent;

import java.lang.instrument.Instrumentation;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.TimeUnit;

public class LoadedClassesCache {
    private static final long CACHE_DURATION = TimeUnit.MINUTES.toMillis(1L);
    private final Instrumentation instrumentation;
    private long lastRefreshTimestamp = 0L;
    private Map<String, Class<?>> allLoadedClassesCache;

    LoadedClassesCache(Instrumentation instrumentation) {
        this.instrumentation = instrumentation;
    }

    public boolean isClassLoaded(String className) {
        return this.findClass(className) != null;
    }

    public synchronized Class<?> findClass(String className) {
        this.updateIfNeeded();
        return this.allLoadedClassesCache.get(className);
    }

    private void updateIfNeeded() {
        long now = System.currentTimeMillis();
        if (this.lastRefreshTimestamp + CACHE_DURATION < now) {
            Class<?>[] loadedClasses = this.instrumentation.getAllLoadedClasses();
            Map<String, Class<?>> cache = new HashMap<>(loadedClasses.length * 2);
            for (Class<?> clazz : loadedClasses) {
                if (clazz != null && !clazz.isArray() && !clazz.isSynthetic()) {
                    if (!clazz.getName().startsWith("com.sun.proxy")) {
                        cache.put(clazz.getName(), clazz);
                    }
                }
            }
            this.allLoadedClassesCache = cache;
            this.lastRefreshTimestamp = now;
        }
    }

    synchronized void check() {
        if (this.lastRefreshTimestamp != 0L && this.lastRefreshTimestamp + CACHE_DURATION < System.currentTimeMillis()) {
            this.clear();
        }
    }

    synchronized void clear() {
        this.lastRefreshTimestamp = 0L;
        if (this.allLoadedClassesCache != null) {
            this.allLoadedClassesCache.clear();
            this.allLoadedClassesCache = null;
        }
    }

}
