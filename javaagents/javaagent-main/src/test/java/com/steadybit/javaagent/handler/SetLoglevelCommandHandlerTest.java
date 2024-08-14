/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.handler;

import com.steadybit.javaagent.CommandHandler;
import com.steadybit.javaagent.log.LogLevel;
import com.steadybit.javaagent.log.RemoteAgentLogger;
import org.junit.jupiter.api.Test;

import java.io.BufferedReader;
import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.InputStreamReader;

import static org.assertj.core.api.Assertions.assertThat;

class SetLoglevelCommandHandlerTest {
    private final CommandHandler handler = new SetLoglevelCommandHandler();

    @Test
    void should_return_true() throws IOException {
        RemoteAgentLogger.setLevel(LogLevel.ERROR);

        //when
        String result = this.command("log-level", "INFO");

        //then
        assertThat(result).isEqualTo("true");

        assertThat(RemoteAgentLogger.getLevel()).isEqualTo(LogLevel.INFO);
    }

    private String command(String command, String argument) throws IOException {
        ByteArrayOutputStream os = new ByteArrayOutputStream();
        this.handler.handle(command, argument, os);
        byte[] buf = os.toByteArray();
        assertThat(buf[0]).isEqualTo(CommandHandler.RC_OK);
        return new BufferedReader(new InputStreamReader(new ByteArrayInputStream(buf, 1, buf.length - 1))).readLine();
    }

}