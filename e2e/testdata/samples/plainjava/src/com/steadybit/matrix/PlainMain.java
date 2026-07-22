/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package com.steadybit.matrix;

import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;

import com.sun.net.httpserver.HttpServer;

/**
 * Minimal non-Spring Java application used to verify the plain-Java attacks
 * (java-method-delay, java-method-exception) across all Java LTS runtimes.
 * Exposes an HTTP endpoint whose latency/outcome reflects a call to the
 * instrumented {@link WorkService#work()} method.
 */
public class PlainMain {

    public static void main(String[] args) throws IOException {
        WorkService workService = new WorkService();
        HttpServer server = HttpServer.create(new InetSocketAddress(8080), 0);
        server.createContext("/work", exchange -> {
            byte[] body;
            int status;
            try {
                body = ("ok:" + workService.work()).getBytes();
                status = 200;
            } catch (RuntimeException e) {
                body = ("error:" + e.getMessage()).getBytes();
                status = 500;
            }
            exchange.sendResponseHeaders(status, body.length);
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(body);
            }
        });
        server.createContext("/health", exchange -> {
            byte[] body = "up".getBytes();
            exchange.sendResponseHeaders(200, body.length);
            try (OutputStream os = exchange.getResponseBody()) {
                os.write(body);
            }
        });
        server.start();
        System.out.println("plainjava sample listening on :8080");
    }
}
