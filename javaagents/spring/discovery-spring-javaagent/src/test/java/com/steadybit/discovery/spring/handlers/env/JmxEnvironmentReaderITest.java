/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.spring.handlers.env;

import com.steadybit.discovery.spring.handlers.EnvCommandHandler;
import com.steadybit.discovery.spring.handlers.TestBootApplication;
import com.steadybit.javaagent.CommandHandler;
import static org.assertj.core.api.Assertions.assertThat;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.web.context.ConfigurableWebServerApplicationContext;

import java.io.ByteArrayOutputStream;
import java.util.Collections;

class JmxEnvironmentReaderITest {
    private ConfigurableWebServerApplicationContext context;
    private CommandHandler handler;

    @BeforeEach
    void setUp() {
        //TODO: Replace with testcontainer and spring-boot-sample?
        this.context = (ConfigurableWebServerApplicationContext) SpringApplication.run(TestBootApplication.class, "--spring.jmx.enabled=true",
                "--server.port=0");
        this.handler = new EnvCommandHandler(Collections::emptyList);
    }

    @AfterEach
    void tearDown() {
        this.context.close();
    }

    @Test
    void should_return_env() {
        String response = this.command("spring-env", "server.port");
        assertThat(response).isEqualTo("0\n");
    }

    @Test
    void should_not_return_env() {
        String response = this.command("spring-env", "does.not.exist");
        assertThat(response).isEqualTo("\n");
    }

    private String command(String command, String arg) {
        ByteArrayOutputStream os = new ByteArrayOutputStream();
        this.handler.handle(command, arg, os);
        byte[] buf = os.toByteArray();
        assertThat(buf[0]).isEqualTo(CommandHandler.RC_OK);
        return new String(buf, 1, buf.length - 1);
    }
}
