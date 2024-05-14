/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.mvc;

import com.steadybit.discovery.springboot.javaagent.handlers.HttpMappingsCommandHandler;
import com.steadybit.discovery.springboot.javaagent.handlers.TestBootApplication;
import com.steadybit.javaagent.CommandHandler;
import org.apache.catalina.Container;
import org.apache.catalina.Context;
import org.apache.catalina.core.StandardWrapper;
import static org.assertj.core.api.Assertions.assertThat;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.Test;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.web.context.ConfigurableWebServerApplicationContext;
import org.springframework.boot.web.embedded.tomcat.TomcatWebServer;

import javax.servlet.ServletException;
import java.io.ByteArrayOutputStream;
import java.io.UnsupportedEncodingException;
import java.util.Collections;

class ApplicationContextMappingDescriptionProviderTest {
    private ConfigurableWebServerApplicationContext context;

    @AfterEach
    void tearDown() {
        this.context.close();
    }

    @Test
    void should_return_mappings() throws UnsupportedEncodingException {
        //TODO: Replace with testcontainer and spring-boot-sample?
        this.context = (ConfigurableWebServerApplicationContext) SpringApplication.run(TestBootApplication.class, "--server.port=0");
        this.initializeDispatcherServlet((TomcatWebServer) this.context.getWebServer(), "dispatcherServlet");

        CommandHandler handler = new HttpMappingsCommandHandler(() -> Collections.singletonList(this.context));

        String response = this.command(handler, "spring-mvc-mappings");
        assertThat(response).contains(
                "{\"handlerClass\":\"com.steadybit.discovery.springboot.javaagent.handlers.TestBootApplication\",\"handlerDescriptor\":\"()Ljava/lang/String;\",\"methods\":[\"GET\"],\"patterns\":[\"/test\"],\"handlerName\":\"test\"}");
    }

    @Test
    void should_return_empty_mappings() throws UnsupportedEncodingException {
        //TODO: Replace with testcontainer and spring-boot-sample?
        this.context = (ConfigurableWebServerApplicationContext) SpringApplication.run(TestBootApplication.class, "--server.port=0");
        this.initializeDispatcherServlet((TomcatWebServer) this.context.getWebServer(), "dispatcherServlet");

        CommandHandler handler = new HttpMappingsCommandHandler(Collections::emptyList);

        String response = this.command(handler, "spring-mvc-mappings");
        assertThat(response).isEqualTo("\ufeff[]");
    }

    @Test
    void should_return_mappings_webflux() throws UnsupportedEncodingException {
        //TODO: Replace with testcontainer and spring-boot-sample?
        this.context = (ConfigurableWebServerApplicationContext) SpringApplication.run(TestBootApplication.class, "--server.port=0", "--spring.main.web-application-type=reactive");
        this.initializeDispatcherServlet((TomcatWebServer) this.context.getWebServer(), "dispatcherServlet");

        CommandHandler handler = new HttpMappingsCommandHandler(() -> Collections.singletonList(this.context));

        String response = this.command(handler, "spring-mvc-mappings");
        assertThat(response).contains(
                "{\"handlerClass\":\"com.steadybit.discovery.springboot.javaagent.handlers.TestBootApplication\",\"handlerDescriptor\":\"()Ljava/lang/String;\",\"methods\":[\"GET\"],\"patterns\":[\"/test\"],\"handlerName\":\"test\"}");
    }

    private String command(CommandHandler handler, String command) {
        ByteArrayOutputStream os = new ByteArrayOutputStream();
        handler.handle(command, "", os);
        byte[] buf = os.toByteArray();
        assertThat(buf[0]).isEqualTo(CommandHandler.RC_OK);
        return new String(buf, 1, buf.length - 1);
    }

    private void initializeDispatcherServlet(TomcatWebServer webServer, String name) {
        for (Container container : webServer.getTomcat().getHost().findChildren()) {
            if (container instanceof Context) {
                Container child = container.findChild(name);
                if (child instanceof StandardWrapper) {
                    try {
                        StandardWrapper wrapper = (StandardWrapper) child;
                        wrapper.deallocate(wrapper.allocate());
                    } catch (ServletException ex) {
                        // Continue
                    }
                }
                return;
            }
        }
    }
}