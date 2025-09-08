/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.log;

import com.github.tomakehurst.wiremock.WireMockServer;
import com.github.tomakehurst.wiremock.client.WireMock;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.io.PrintWriter;
import java.io.StringWriter;
import java.net.MalformedURLException;
import java.net.URI;

import static com.github.tomakehurst.wiremock.client.WireMock.anyRequestedFor;
import static com.github.tomakehurst.wiremock.client.WireMock.equalTo;
import static com.github.tomakehurst.wiremock.client.WireMock.postRequestedFor;
import static com.github.tomakehurst.wiremock.client.WireMock.urlEqualTo;
import static com.github.tomakehurst.wiremock.core.WireMockConfiguration.options;

class RemoteAgentLoggerTest {
    private static WireMockServer wireMock;

    @BeforeAll
    static void beforeAll() throws MalformedURLException {
        wireMock = new WireMockServer(options().dynamicPort());
        wireMock.start();
        RemoteAgentLogger.init("2", URI.create(wireMock.url("/log")).toURL());
        RemoteAgentLogger.setLevel(LogLevel.ERROR);
    }

    @AfterAll
    static void afterAll() {
        wireMock.stop();
    }

    @BeforeEach
    void setUp() {
        wireMock.resetAll();
    }

    @Test
    void should_not_send_log() {
        RemoteAgentLogger.getLogger(Object.class).debug("This is  debug message");

        wireMock.verify(0, anyRequestedFor(urlEqualTo("/log")));
    }

    @Test
    void should_send_log() {
        wireMock.stubFor(WireMock.post("/log").willReturn(WireMock.aResponse().withStatus(200)));

        RuntimeException exception = new RuntimeException("test");
        RemoteAgentLogger.getLogger(Object.class).error("This is a Message with stacktrace", exception);

        String stacktrace = this.stacktracetoString(exception);
        String expectedBody = "{\"msg\":\"This is a Message with stacktrace\",\"pid\":\"2\",\"level\":\"ERROR\",\"stacktrace\":\"" + stacktrace + "\"}";
        wireMock.verify(postRequestedFor(urlEqualTo("/log")).withHeader("Content-Type", equalTo("application/json"))
                .withHeader("Content-Length", equalTo(Integer.toString(expectedBody.length())))
                .withRequestBody(equalTo(expectedBody)));
    }

    private String stacktracetoString(RuntimeException exception) {
        StringWriter sw = new StringWriter();
        exception.printStackTrace(new PrintWriter(sw));
        return sw.toString().replace("\t", "\\t").replace("\n", "\\n").replace("/", "\\/");
    }
}