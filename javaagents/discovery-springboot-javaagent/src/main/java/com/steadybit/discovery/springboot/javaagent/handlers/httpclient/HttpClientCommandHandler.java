/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.httpclient;

import com.steadybit.javaagent.CommandHandler;
import org.json.JSONArray;
import org.json.JSONObject;

import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.io.PrintWriter;
import java.nio.charset.StandardCharsets;
import java.util.Collection;
import java.util.function.Supplier;
import java.util.stream.Collectors;

public class HttpClientCommandHandler implements CommandHandler {
    private static final char BYTE_ORDER_MARK = '\ufeff';

    private final Supplier<Collection<HttpRequest>> requestSupplier;

    public HttpClientCommandHandler(Supplier<Collection<HttpRequest>> requestSupplier) {
        this.requestSupplier = requestSupplier;
    }

    @Override
    public boolean canHandle(String command) {
        return command.equals("spring-httpclient-requests") || command.equals("spring-httpclient-addresses");
    }

    @Override
    public void handle(String command, String argument, OutputStream os) {
        JSONArray json;
        if (command.equals("spring-httpclient-addresses")) {
            json = new JSONArray(this.requestSupplier.get().stream().map(HttpRequest::getAddress).collect(Collectors.toList()));
        } else {
            json = new JSONArray(this.requestSupplier.get().stream().map(this::toJson).collect(Collectors.toList()));
        }

        PrintWriter writer = new PrintWriter(new OutputStreamWriter(os, StandardCharsets.UTF_8));
        writer.write(RC_OK);
        writer.write(BYTE_ORDER_MARK);
        json.write(writer);
        writer.flush();
    }

    private JSONObject toJson(HttpRequest request) {
        JSONObject json = new JSONObject();
        json.put("address", request.getAddress());
        json.put("scheme", request.getScheme());
        json.put("circuitBreaker", request.hasCircuitBreaker());
        json.put("timeout", request.getTimeout());
        return json;
    }
}
