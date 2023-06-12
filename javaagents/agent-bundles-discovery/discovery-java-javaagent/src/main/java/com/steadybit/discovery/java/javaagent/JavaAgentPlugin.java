/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.java.javaagent;

import com.steadybit.discovery.java.javaagent.handlers.DataSourceCommandHandler;
import com.steadybit.discovery.java.javaagent.handlers.datasource.DataSourceScanner;
import com.steadybit.javaagent.AgentPlugin;
import com.steadybit.javaagent.CommandHandler;

import java.io.OutputStream;
import java.lang.instrument.Instrumentation;
import java.util.Collections;
import java.util.List;

/**
 * AgentPlugin to discover data in a Java application
 */
public class JavaAgentPlugin implements AgentPlugin, CommandHandler {
    private final List<CommandHandler> commandHandlers;
    private final DataSourceScanner dataSourceScanner;

    public JavaAgentPlugin(Instrumentation instrumentation) {
        this.dataSourceScanner = new DataSourceScanner(instrumentation);
        this.commandHandlers = Collections.singletonList(new DataSourceCommandHandler(this.dataSourceScanner::getDataSourceConnections));
    }

    @Override
    public void start() {
        this.dataSourceScanner.install();
    }

    @Override
    public void destroy() {
        this.dataSourceScanner.reset();
    }

    @Override
    public boolean canHandle(String command) {
        for (CommandHandler commandHandler : this.commandHandlers) {
            if (commandHandler.canHandle(command)) {
                return true;
            }
        }
        return false;
    }

    @Override
    public void handle(String command, String argument, OutputStream os) {
        for (CommandHandler commandHandler : this.commandHandlers) {
            if (commandHandler.canHandle(command)) {
                commandHandler.handle(command, argument, os);
                return;
            }
        }
    }
}
