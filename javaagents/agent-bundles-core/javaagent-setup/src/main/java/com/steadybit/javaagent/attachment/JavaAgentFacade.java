/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment;

import org.springframework.core.io.Resource;

import java.io.InputStream;
import java.util.function.BiFunction;

public interface JavaAgentFacade {

    boolean setLogLevel(JavaVm jvm, String logLevel);

    boolean hasClassLoaded(JavaVm jvm, String className);

    boolean loadAgentPlugin(JavaVm jvm, Resource pluginUri, String args);

    boolean hasAgentPlugin(JavaVm jvm, Resource pluginUri);

    boolean unloadAgentPlugin(JavaVm jvm, Resource pluginUri);

    boolean sendCommandToAgent(JavaVm jvm, String command, String args);

    <T> T sendCommandToAgent(JavaVm jvm, String command, String args, BiFunction<InputStream, CommandResult, T> handler);

    String getAgentHost(JavaVm vm);

    boolean isAttached(JavaVm vm);

    void addAutoloadAgentPlugin(Resource pluginUri, String markerClass);

    void removeAutoloadAgentPlugin(Resource pluginUri, String markerClass);
}
