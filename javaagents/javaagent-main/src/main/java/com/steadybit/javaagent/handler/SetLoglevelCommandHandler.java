/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.handler;

import com.steadybit.javaagent.CommandHandler;
import com.steadybit.javaagent.log.LogLevel;
import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;

import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.io.PrintWriter;
import java.nio.charset.StandardCharsets;

public class SetLoglevelCommandHandler implements CommandHandler {
    private static final Logger log = RemoteAgentLogger.getLogger(SetLoglevelCommandHandler.class);

    @Override
    public boolean canHandle(String command) {
        return "log-level".equals(command);
    }

    @Override
    public void handle(String command, String argument, OutputStream os) {
        RemoteAgentLogger.setLevel(LogLevel.valueOf(argument));
        log.debug(String.format("Set loglevel to %s.", argument));
        PrintWriter writer = new PrintWriter(new OutputStreamWriter(os, StandardCharsets.UTF_8), true);
        writer.write(RC_OK);
        writer.println(Boolean.TRUE);
    }
}
