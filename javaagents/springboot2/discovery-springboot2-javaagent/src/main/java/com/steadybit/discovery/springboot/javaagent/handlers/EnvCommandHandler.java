/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers;

import com.steadybit.discovery.springboot.javaagent.handlers.env.ApplicationContextEnvironmentReader;
import com.steadybit.discovery.springboot.javaagent.handlers.env.JmxEnvironmentReader;
import com.steadybit.javaagent.CommandHandler;
import org.springframework.context.ApplicationContext;

import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.io.PrintWriter;
import java.nio.charset.StandardCharsets;
import java.util.Collection;
import java.util.function.Supplier;

public class EnvCommandHandler implements CommandHandler {
    private final ApplicationContextEnvironmentReader applicationContextEnvironmentReader;
    private final JmxEnvironmentReader jmxEnvironmentReader;

    public EnvCommandHandler(Supplier<Collection<ApplicationContext>> applicationContextProvider) {
        this.applicationContextEnvironmentReader = new ApplicationContextEnvironmentReader(applicationContextProvider);
        this.jmxEnvironmentReader = new JmxEnvironmentReader();
    }

    @Override
    public boolean canHandle(String command) {
        return command.equals("spring-env");
    }

    @Override
    public void handle(String command, String argument, OutputStream os) {
        String value = this.jmxEnvironmentReader.readPropertyValue(argument);
        if (value == null) {
            value = this.applicationContextEnvironmentReader.readPropertyValue(argument);
        }
        PrintWriter writer = new PrintWriter(new OutputStreamWriter(os, StandardCharsets.UTF_8), true);
        writer.write(RC_OK);
        writer.println(value != null ? value : "");
    }
}
