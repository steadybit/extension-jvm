/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent;

import com.steadybit.discovery.springboot.javaagent.handlers.BeanCommandHandler;
import com.steadybit.discovery.springboot.javaagent.handlers.HttpClientCommandHandler;
import com.steadybit.discovery.springboot.javaagent.handlers.HttpMappingsCommandHandler;
import com.steadybit.discovery.springboot.javaagent.handlers.httpclient.HttpClientRequestScanner;
import com.steadybit.javaagent.AgentPlugin;
import com.steadybit.javaagent.CommandHandler;

import java.io.OutputStream;
import java.lang.instrument.Instrumentation;
import java.util.Arrays;
import java.util.List;

/**
 * AgentPlugin to discover data in a spring boot application
 */
public class SpringBootAgentPlugin implements AgentPlugin, CommandHandler {
    private final List<CommandHandler> commandHandlers;
    private final HttpClientRequestScanner httpClientRequestScanner;

    public SpringBootAgentPlugin(Instrumentation instrumentation) {
        this.httpClientRequestScanner = new HttpClientRequestScanner(instrumentation);
        this.commandHandlers = Arrays.asList(new HttpMappingsCommandHandler(), new BeanCommandHandler(), new HttpClientCommandHandler(this.httpClientRequestScanner::getRequests));
    }

    @Override
    public void start() {
        this.httpClientRequestScanner.install();
    }

    @Override
    public void destroy() {
        this.httpClientRequestScanner.reset();
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
