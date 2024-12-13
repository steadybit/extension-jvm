/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent;

import com.steadybit.javaagent.handler.ClassLoadedCommandHandler;
import com.steadybit.javaagent.handler.LoadAgentPluginCommandHandler;
import com.steadybit.javaagent.handler.SetLoglevelCommandHandler;
import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;
import com.steadybit.javaagent.util.CountingOutputStream;

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.io.PrintWriter;
import java.lang.instrument.Instrumentation;
import java.net.Socket;
import java.nio.charset.StandardCharsets;
import java.util.Arrays;
import java.util.List;

/**
 * Accept handler for incoming tcp connections. The actual handling of the connection is delegated to the CommandHandler implementations.
 */
public class JavaAgentSocketHandler {
    private static final Logger log = RemoteAgentLogger.getLogger(JavaAgentSocketHandler.class);
    private final List<CommandHandler> commandHandlers;
    private final LoadedClassesCache loadedClassesCache;

    public JavaAgentSocketHandler(Instrumentation instrumentation) {
        this.loadedClassesCache = new LoadedClassesCache(instrumentation);
        this.commandHandlers = Arrays.asList(
                new ClassLoadedCommandHandler(this.loadedClassesCache),
                new LoadAgentPluginCommandHandler(instrumentation, this.loadedClassesCache),
                new SetLoglevelCommandHandler()
        );
    }


    public void accept(Socket socket) throws IOException {
        OutputStream os = socket.getOutputStream();
        BufferedReader br = new BufferedReader(new InputStreamReader(socket.getInputStream()));

        while (true) {
            String command = br.readLine();
            if (command == null) {
                return;
            }

            this.handleCommand(command, os);
        }
    }

    public void disconnected() {
        this.loadedClassesCache.evict();
    }

    private void handleCommand(String line, OutputStream os) {
        int commandSeparator = line.indexOf(":");
        PrintWriter pw = new PrintWriter(new OutputStreamWriter(os, StandardCharsets.UTF_8), true);
        if (commandSeparator == -1) {
            pw.write(CommandHandler.RC_ERROR);
            pw.println("Invalid command: " + line);
        } else {
            log.trace("Received command: {}", line);
            String command = line.substring(0, commandSeparator);
            String argument = line.substring(commandSeparator + 1);

            for (CommandHandler commandHandler : this.commandHandlers) {
                if (commandHandler.canHandle(command)) {
                    CountingOutputStream countedOs = new CountingOutputStream(os);
                    try {
                        commandHandler.handle(command, argument, countedOs);
                    } catch (Throwable e) {
                        if (log.isDebugEnabled()) {
                            log.warn(String.format("Unexpected Exception running command '%s:%s'", command, argument), e);
                        } else {
                            log.warn("Unexpected Exception running command '{}:{}': {} - {}", command, argument, e.getClass().getSimpleName(), e.getMessage());
                        }
                        if (countedOs.getCount() == 0L) {
                            try {
                                os.write(CommandHandler.RC_ERROR);
                            } catch (IOException ex) {
                                //ignore when error can't be reported
                            }
                        }
                    }

                    if (countedOs.getCount() == 0L) {
                        try {
                            os.write(CommandHandler.RC_OK);
                        } catch (IOException ex) {
                            //ignore when error can't be reported
                        }
                    }
                    return;
                }
            }

            pw.write(CommandHandler.RC_ERROR);
            pw.println("Unknown command: " + command);
            log.warn("Received unknown command: {}", command);
        }
    }
}
