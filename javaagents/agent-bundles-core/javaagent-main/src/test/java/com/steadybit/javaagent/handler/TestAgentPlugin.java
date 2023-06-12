/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.handler;

import com.steadybit.javaagent.AgentPlugin;

public class TestAgentPlugin implements AgentPlugin {

    @Override
    public void start() {
        System.err.println("TestAgentPlugin loaded");
        LoadAgentPluginCommandHandlerTest.pluginLoaded = true;
    }

    @Override
    public void destroy() {
        System.err.println("TestAgentPlugin unloaded");
        LoadAgentPluginCommandHandlerTest.pluginUnloaded = true;
    }
}