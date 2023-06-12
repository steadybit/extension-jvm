/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers;

import com.steadybit.discovery.springboot.javaagent.handlers.beans.ApplicationContextBeanReader;
import com.steadybit.discovery.springboot.javaagent.handlers.beans.JmxBeanReader;
import com.steadybit.javaagent.CommandHandler;
import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;
import org.springframework.context.ApplicationContext;

import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.io.PrintWriter;
import java.nio.charset.StandardCharsets;
import java.util.Collection;
import java.util.function.Supplier;

public class BeanCommandHandler implements CommandHandler {
    private static final Logger log = RemoteAgentLogger.getLogger(BeanCommandHandler.class);
    private final ApplicationContextBeanReader applicationContextBeanReader;
    private final JmxBeanReader jmxBeanReader;

    public BeanCommandHandler(Supplier<Collection<ApplicationContext>> applicationContextProvider) {
        this.applicationContextBeanReader = new ApplicationContextBeanReader(applicationContextProvider);
        this.jmxBeanReader = new JmxBeanReader();
    }

    @Override
    public boolean canHandle(String command) {
        return command.equals("spring-bean");
    }

    @Override
    public void handle(String command, String argument, OutputStream os) {
        boolean result = this.hasBeanOfType(argument);
        PrintWriter writer = new PrintWriter(new OutputStreamWriter(os, StandardCharsets.UTF_8), true);
        writer.write(RC_OK);
        writer.println(result);
    }

    private boolean hasBeanOfType(String className) {
        try {
            Class<?> beanClass = Class.forName(className);
            Boolean value = this.jmxBeanReader.hasBeanOfType(beanClass);
            if (value == null) {
                value = this.applicationContextBeanReader.hasBeanOfType(beanClass);
            }
            return Boolean.TRUE.equals(value);
        } catch (ClassNotFoundException e) {
            log.trace("Could not find class " + className + " when searching for bean: " + e.getMessage());
            return false;
        }
    }
}
