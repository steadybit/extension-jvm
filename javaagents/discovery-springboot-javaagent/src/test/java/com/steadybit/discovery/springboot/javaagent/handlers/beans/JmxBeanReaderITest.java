/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.beans;

import com.steadybit.discovery.springboot.javaagent.handlers.BeanCommandHandler;
import com.steadybit.discovery.springboot.javaagent.handlers.TestBootApplication;
import com.steadybit.javaagent.CommandHandler;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.web.context.ConfigurableWebServerApplicationContext;

import java.io.ByteArrayOutputStream;
import java.util.Collections;

import static org.assertj.core.api.Assertions.assertThat;

class JmxBeanReaderITest {
    private ConfigurableWebServerApplicationContext context;
    private CommandHandler handler;

    @BeforeEach
    void setUp() {
        //TODO: Replace with testcontainer and spring-boot-sample?
        this.context = (ConfigurableWebServerApplicationContext) SpringApplication.run(TestBootApplication.class, "--spring.jmx.enabled=true",
                "--server.port=0");
        this.handler = new BeanCommandHandler();
    }

    @AfterEach
    void tearDown() {
        this.context.close();
    }

    @Test
    void should_return_bean() {
        String response = this.command("spring-bean", "com.steadybit.discovery.springboot.javaagent.handlers.TestBootApplication");
        assertThat(response).isEqualTo("true\n");
    }

    @Test
    void should_not_return_bean() {
        String response = this.command("spring-bean", "java.util.List");
        assertThat(response).isEqualTo("false\n");
    }

    @Test
    void should_not_return_bean_invalid_class() {
        String response = this.command("spring-bean", "class.does.not.exist");
        assertThat(response).isEqualTo("false\n");
    }

    @Test
    void should_return_main_context_name() {
        String response = this.command("spring-main-context","");
        assertThat(response).isEqualTo("application\n");
    }

    private String command(String command, String arg) {
        ByteArrayOutputStream os = new ByteArrayOutputStream();
        this.handler.handle(command, arg, os);
        byte[] buf = os.toByteArray();
        assertThat(buf[0]).isEqualTo(CommandHandler.RC_OK);
        return new String(buf, 1, buf.length - 1);
    }
}