/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.instrumentation;

import java.util.Arrays;

public class InstrumentationPluginDispatcher {
    static final InstrumentationPlugin[] plugins = new InstrumentationPlugin[512];
    private static int next;

    static {
        reset();
    }

    public static InstrumentationPlugin find(int registration) {
        return plugins[registration];
    }

    public static void register(InstrumentationPlugin plugin) {
        synchronized (plugins) {
            if (plugin == null) {
                throw new IllegalArgumentException("plugin must not be null");
            }
            if (plugins[next] != InstrumentationPlugin.NOOP) {
                throw new IllegalStateException(
                        "Registration " + next + " is already used by plugin " + plugins[next] + ", but " + plugin + " tries to register it");
            } else {
                plugins[next] = plugin;
                plugin.setRegistration(next);
                while (next < plugins.length && plugins[next] != InstrumentationPlugin.NOOP) {
                    next++;
                }
            }
        }
    }

    public static void deregister(InstrumentationPlugin plugin) {
        int registration = plugin.getRegistration();
        if (registration >= 0) {
            synchronized (plugins) {
                if (plugins[registration] == plugin) {
                    plugins[registration] = InstrumentationPlugin.NOOP;
                    next = registration;
                }
                plugin.setRegistration(-1);
            }
        }
    }

    static void reset() {
        synchronized (plugins) {
            next = 0;
            Arrays.fill(plugins, InstrumentationPlugin.NOOP);
        }
    }
}
