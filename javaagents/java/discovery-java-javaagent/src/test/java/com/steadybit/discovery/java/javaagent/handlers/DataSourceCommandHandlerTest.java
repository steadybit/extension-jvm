/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.java.javaagent.handlers;

import com.steadybit.discovery.java.javaagent.handlers.datasource.DataSourceConnection;
import com.steadybit.javaagent.CommandHandler;
import static org.assertj.core.api.Assertions.assertThat;
import org.junit.jupiter.api.Test;

import java.io.ByteArrayOutputStream;
import java.util.Collections;

class DataSourceCommandHandlerTest {
    @Test
    void should_return_datasources() {
        DataSourceConnection connection = new DataSourceConnection("jdbc:test", "test");
        CommandHandler handler = new DataSourceCommandHandler(() -> Collections.singletonList(connection));

        String response = this.command(handler, "java-datasource-connection");
        assertThat(response).isEqualTo("\uFEFF[{\"databaseType\":\"test\",\"jdbcUrl\":\"jdbc:test\"}]");
    }

    @Test
    void should_return_empty_datasources() {
        CommandHandler handler = new DataSourceCommandHandler(Collections::emptyList);

        String response = this.command(handler, "java-datasource-connection");
        assertThat(response).isEqualTo("\uFEFF[]");
    }

    private String command(CommandHandler handler, String command) {
        ByteArrayOutputStream os = new ByteArrayOutputStream();
        handler.handle(command, "", os);
        byte[] buf = os.toByteArray();
        assertThat(buf[0]).isEqualTo(CommandHandler.RC_OK);
        return new String(buf, 1, buf.length - 1);
    }
}