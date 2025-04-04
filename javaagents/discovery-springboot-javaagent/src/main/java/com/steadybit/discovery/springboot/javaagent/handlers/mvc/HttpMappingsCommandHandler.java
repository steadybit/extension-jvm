/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers;

import com.steadybit.discovery.springboot.javaagent.handlers.mvc.JmxMappingDescriptionProvider;
import com.steadybit.javaagent.CommandHandler;
import org.json.JSONArray;
import org.springframework.context.ApplicationContext;

import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.io.PrintWriter;
import java.nio.charset.StandardCharsets;
import java.util.Collection;
import java.util.function.Supplier;

public class HttpMappingsCommandHandler implements CommandHandler {
    private static final char BYTE_ORDER_MARK = '\ufeff';
    private final JmxMappingDescriptionProvider jmxMappingDescriptionProvider;

    public HttpMappingsCommandHandler() {
        this.jmxMappingDescriptionProvider = new JmxMappingDescriptionProvider();
    }

    @Override
    public boolean canHandle(String command) {
        return command.equals("spring-mvc-mappings");
    }

    @Override
    public void handle(String command, String argument, OutputStream os) {
        JSONArray mappings = new JSONArray();

        this.jmxMappingDescriptionProvider.describeMappings(mappings);

        PrintWriter writer = new PrintWriter(new OutputStreamWriter(os, StandardCharsets.UTF_8));
        writer.write(RC_OK);
        writer.write(BYTE_ORDER_MARK);
        mappings.write(writer);
        writer.flush();
    }
}
