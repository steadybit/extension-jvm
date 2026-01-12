/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.mvc;

import com.steadybit.discovery.springboot.javaagent.handlers.TestBootApplication;
import com.steadybit.javaagent.CommandHandler;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Test;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.web.context.ConfigurableWebServerApplicationContext;

import java.io.ByteArrayOutputStream;
import java.io.UnsupportedEncodingException;

import static org.assertj.core.api.Assertions.assertThat;

class JmxMappingDescriptionProviderITest {
    private ConfigurableWebServerApplicationContext context;
    private CommandHandler handler = new HttpMappingsCommandHandler();

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
                "{\"handlerClass\":\"com.steadybit.discovery.springboot.javaagent.handlers.TestBootApplication\",\"handlerDescriptor\":\"()Ljava/lang/String;\",\"methods\":[\"GET\"],\"patterns\":[\"/test\"],\"handlerName\":\"test\"}");
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
                "{\"handlerClass\":\"com.steadybit.discovery.springboot.javaagent.handlers.TestBootApplication\",\"handlerDescriptor\":\"()Ljava/lang/String;\",\"methods\":[\"GET\"],\"patterns\":[\"/test\"],\"handlerName\":\"test\"}");
    }
}