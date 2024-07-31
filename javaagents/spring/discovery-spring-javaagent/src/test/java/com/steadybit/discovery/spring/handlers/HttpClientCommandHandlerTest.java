/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.spring.handlers;

import com.steadybit.discovery.spring.handlers.httpclient.HttpRequest;
import com.steadybit.javaagent.CommandHandler;
import static org.assertj.core.api.Assertions.assertThat;
import org.junit.jupiter.api.Test;

import java.io.ByteArrayOutputStream;
import java.util.Arrays;
import java.util.Collections;

class HttpClientCommandHandlerTest {
    @Test
    void should_return_addresses() {
        CommandHandler handler = new HttpClientCommandHandler(
                () -> Arrays.asList(new HttpRequest("localhost", "http", false, 0), new HttpRequest("google.com:443", "https", true, 250)));

        String response = this.command(handler, "spring-httpclient-addresses");

        assertThat(response).contains("[\"localhost\",\"google.com:443\"]");
    }

    @Test
    void should_return_empty_addresses() {
        CommandHandler handler = new HttpClientCommandHandler(Collections::emptyList);

        String response = this.command(handler, "spring-httpclient-addresses");
        assertThat(response).isEqualTo("\ufeff[]");
    }

    @Test
    void should_return_requests() {
        CommandHandler handler = new HttpClientCommandHandler(
                () -> Arrays.asList(new HttpRequest("localhost", "http", false, 0), new HttpRequest("google.com:443", "https", true, 250)));

        String response = this.command(handler, "spring-httpclient-requests");
        assertThat(response).contains(
                "[{\"address\":\"localhost\",\"scheme\":\"http\",\"circuitBreaker\":false,\"timeout\":0},{\"address\":\"google.com:443\",\"scheme\":\"https\",\"circuitBreaker\":true,\"timeout\":250}]");
    }

    @Test
    void should_return_empty_requests() {
        CommandHandler handler = new HttpClientCommandHandler(Collections::emptyList);

        String response = this.command(handler, "spring-httpclient-requests");
        assertThat(response).isEqualTo("\ufeff[]");
    }

    private String command(CommandHandler handler, String command) {
        ByteArrayOutputStream os = new ByteArrayOutputStream();
        handler.handle(command, "", os);
        byte[] buf = os.toByteArray();
        assertThat(buf[0]).isEqualTo(CommandHandler.RC_OK);
        return new String(buf, 1, buf.length - 1);
    }
}
