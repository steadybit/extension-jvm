/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.handler;

import com.steadybit.javaagent.CommandHandler;
import com.steadybit.javaagent.LoadedClassesCache;

import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.io.PrintWriter;
import java.nio.charset.StandardCharsets;

public class ClassLoadedCommandHandler implements CommandHandler {
    private final LoadedClassesCache loadedClassesCache;

    public ClassLoadedCommandHandler(LoadedClassesCache loadedClassesCache) {
        this.loadedClassesCache = loadedClassesCache;
    }

    @Override
    public boolean canHandle(String command) {
        return "class-loaded".equals(command);
    }

    @Override
    public void handle(String command, String argument, OutputStream os) {
        PrintWriter writer = new PrintWriter(new OutputStreamWriter(os, StandardCharsets.UTF_8), true);
        writer.write(RC_OK);
        writer.println(this.isClassLoaded(argument));
    }

    private boolean isClassLoaded(String className) {
        return this.loadedClassesCache.isClassLoaded(className);
    }
}
