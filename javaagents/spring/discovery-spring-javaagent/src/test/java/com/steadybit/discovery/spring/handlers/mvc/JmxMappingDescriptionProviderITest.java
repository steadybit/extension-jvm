/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.spring.handlers.mvc;

import com.steadybit.discovery.spring.handlers.HttpMappingsCommandHandler;
import com.steadybit.discovery.spring.handlers.TestBootApplication;
import com.steadybit.javaagent.CommandHandler;
import static org.assertj.core.api.Assertions.assertThat;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.web.context.ConfigurableWebServerApplicationContext;

import java.io.ByteArrayOutputStream;
import java.io.UnsupportedEncodingException;
import java.util.Collections;

class JmxMappingDescriptionProviderITest {
    private ConfigurableWebServerApplicationContext context;
    private CommandHandler handler;

    @BeforeEach
    void setUp() {
        this.handler = new HttpMappingsCommandHandler(Collections::emptyList);
    }

    @AfterEach
    void tearDown() {
        this.context.close();
    }

    @Test
    void should_return_mappings() throws UnsupportedEncodingException {
        this.context = (ConfigurableWebServerApplicationContext) SpringApplication.run(
                TestBootApplication.class,
                "--spring.jmx.enabled=true",
                "--server.port=0");

        ByteArrayOutputStream os = new ByteArrayOutputStream();
        this.handler.handle("spring-mvc-mappings", null, os);
        String response = os.toString("UTF-8");
        assertThat(response).contains(
                "{\"handlerClass\":\"com.steadybit.discovery.spring.handlers.TestBootApplication\",\"handlerDescriptor\":\"()Ljava/lang/String;\",\"methods\":[\"GET\"],\"patterns\":[\"/test\"],\"handlerName\":\"test\"}");
    }

    @Test
    void should_return_mappings_for_webflux() throws UnsupportedEncodingException {
        this.context = (ConfigurableWebServerApplicationContext) SpringApplication.run(
                TestBootApplication.class,
                "--spring.jmx.enabled=true",
                "--spring.main.web-application-type=reactive",
                "--server.port=0");

        ByteArrayOutputStream os = new ByteArrayOutputStream();
        this.handler.handle("spring-mvc-mappings", null, os);
        String response = os.toString("UTF-8");
        assertThat(response).contains(
                "{\"handlerClass\":\"com.steadybit.discovery.spring.handlers.TestBootApplication\",\"handlerDescriptor\":\"()Ljava/lang/String;\",\"methods\":[\"GET\"],\"patterns\":[\"/test\"],\"handlerName\":\"test\"}");
    }
}
