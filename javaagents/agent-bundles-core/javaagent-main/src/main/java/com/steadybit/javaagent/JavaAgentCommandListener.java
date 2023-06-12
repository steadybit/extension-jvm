/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent;

import com.steadybit.javaagent.handler.AppliedAutofixesCommandHandler;
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
import java.net.HttpURLConnection;
import java.net.MalformedURLException;
import java.net.Proxy;
import java.net.ServerSocket;
import java.net.Socket;
import java.net.SocketTimeoutException;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.security.AccessController;
import java.security.PrivilegedAction;
import java.util.Arrays;
import java.util.List;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;

/**
 * Main Javaagent Thread.
 * Opens a server socket to receive commands to execute and announces itself at the steadybit agent using http.
 * The command exection is delegated to the CommandHandler implementations.
 */
public class JavaAgentCommandListener extends Thread {
    private static final Logger log = RemoteAgentLogger.getLogger(JavaAgentCommandListener.class);
    private static final int SOCKET_ACCEPT_TIMEOUT = (int) TimeUnit.SECONDS.toMillis(60L);
    private static final String THREADNAME_PREFIX = "steadybit-java-agent";
    private static final int REGISTER_CONNECT_TIMEOUT = 2000;
    private static final int REGISTER_READ_TIMEOUT = 5_000;
    private static final long REGISTER_BACKOFF = 1000L;
    private static final long REGISTER_BACKOFF_MAX = 15_000L;
    private static final long REGISTER_BACKOFF_MULTIPLIER = 2L;

    private final LoadAgentPluginCommandHandler loadAgentPluginCommandHandler;
    private final String pid;
    private final String agentHost;
    private final AtomicBoolean connected = new AtomicBoolean(false);
    private final AtomicBoolean shutdown = new AtomicBoolean(false);
    private final URL registerUrl;
    private final List<CommandHandler> commandHandlers;
    private final LoadedClassesCache loadedClassesCache;
    private ServerSocket serverSocket;
    private String remoteAddress;

    public JavaAgentCommandListener(String pid, String agentHost, String agentPort, Instrumentation instrumentation) throws MalformedURLException {
        super(THREADNAME_PREFIX);
        this.pid = pid;
        this.agentHost = agentHost;
        this.registerUrl = new URL(String.format("http://%s:%s/javaagent", this.agentHost, agentPort));
        this.loadedClassesCache = new LoadedClassesCache(instrumentation);
        this.loadAgentPluginCommandHandler = new LoadAgentPluginCommandHandler(instrumentation, this.loadedClassesCache);
        this.commandHandlers = Arrays.asList(new ClassLoadedCommandHandler(this.loadedClassesCache), this.loadAgentPluginCommandHandler,
                new SetLoglevelCommandHandler(), new AppliedAutofixesCommandHandler());
    }

    @Override
    public void run() {
        try {
            AccessController.doPrivileged((PrivilegedAction<Void>) () -> {
                try {
                    JavaAgentCommandListener.this.serverSocket = new ServerSocket(0, 0);
                    if ("127.0.0.1".equals(JavaAgentCommandListener.this.agentHost)) {
                        JavaAgentCommandListener.this.remoteAddress = String.format("%s=%s:%s", JavaAgentCommandListener.this.pid, "127.0.0.1",
                                JavaAgentCommandListener.this.serverSocket.getLocalPort());
                    } else {
                        JavaAgentCommandListener.this.remoteAddress = String.format("%s=%s", JavaAgentCommandListener.this.pid,
                                JavaAgentCommandListener.this.serverSocket.getLocalPort());
                    }
                    log.debug(String.format("Created ServerSocket with remote address %s", JavaAgentCommandListener.this.remoteAddress));
                    JavaAgentCommandListener.this.serverSocket.setSoTimeout(JavaAgentCommandListener.SOCKET_ACCEPT_TIMEOUT);
                    JavaAgentCommandListener.this.listen();
                } catch (Exception e) {
                    log.error("Uncaught exception", e);
                } finally {
                    JavaAgentCommandListener.this.shutdown();
                }
                return null;
            });
        } catch (Throwable e) {
            log.error("Could not start JavaAgentThread", e);
            this.shutdown();
        }
    }

    private void listen() {
        this.announce();
        do {
            try {
                this.loadedClassesCache.check();
                this.accept();
            } catch (SocketTimeoutException ex) {
                log.trace("Could not accept remote connection: " + ex.getMessage());
            } catch (IOException ex) {
                if (!this.shutdown.get()) {
                    log.debug("Could not accept remote connection: " + ex.getMessage());
                }
            } catch (Exception ex) {
                log.debug("Exception while listening for commands: ", ex);
            }
        } while (!this.shutdown.get());
    }

    private void accept() throws IOException {
        OutputStream os = null;
        BufferedReader br = null;
        Socket socket = null;
        try {
            log.trace(
                    String.format("Accepting data for JVM with PID %s on %s:%s", this.pid,
                            JavaAgentCommandListener.this.serverSocket.getInetAddress().getHostAddress(),
                            JavaAgentCommandListener.this.serverSocket.getLocalPort()));
            socket = this.serverSocket.accept();
            os = socket.getOutputStream();
            br = new BufferedReader(new InputStreamReader(socket.getInputStream()));

            while (!this.shutdown.get()) {
                String command = br.readLine();
                if (command == null) {
                    return;
                }

                log.trace(String.format("Received command: %s ", command));
                this.handleCommand(command, os);
            }
        } finally {
            if (os != null) {
                try {
                    os.close();
                } catch (IOException e) {
                    log.warn("Could not close OutputStream:" + e.getMessage());
                }
            }
            if (br != null) {
                try {
                    br.close();
                } catch (IOException e) {
                    log.warn("Could not close BufferedReader:" + e.getMessage());
                }
            }
            if (socket != null) {
                try {
                    socket.close();
                } catch (IOException e) {
                    log.warn("Could not close Socket:" + e.getMessage());
                }
            }
        }
    }

    private void handleCommand(String line, OutputStream os) {
        int commandSeparator = line.indexOf(":");
        PrintWriter pw = new PrintWriter(new OutputStreamWriter(os, StandardCharsets.UTF_8), true);
        if (commandSeparator == -1) {
            pw.write(CommandHandler.RC_ERROR);
            pw.println("Invalid command: " + line);
        } else {
            String command = line.substring(0, commandSeparator);
            String argument = line.substring(commandSeparator + 1);
            log.trace("Received command: {}", command);

            for (CommandHandler commandHandler : this.commandHandlers) {
                if (commandHandler.canHandle(command)) {
                    CountingOutputStream countedOs = new CountingOutputStream(os);
                    try {
                        commandHandler.handle(command, argument, countedOs);
                    } catch (Throwable e) {
                        log.warn("Unexpected Exception running command '{}:{}': {}", command, argument, e.getMessage());
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

    private void announce() {
        int attempts = 0;
        while (!this.shutdown.get()) {
            HttpResponse response = this.registerAgent(this.registerUrl, this.remoteAddress.getBytes(StandardCharsets.UTF_8));
            if (response != null && response.code >= 200 && response.code <= 299) {
                log.debug(String.format("Agent successfully registered to %s with HTTP Code %s", this.registerUrl, response.code));
                break;
            } else {
                log.debug(String.format("Agent failed register with HTTP Code %s", response != null ? response.code : null));
            }

            if (attempts++ > 60 && this.connected.getAndSet(false)) {
                log.error(String.format("Lost connection to %s", this.registerUrl));
                this.lostConnection();
            }

            long backoff = Math.min(REGISTER_BACKOFF_MAX, (long) (Math.pow(REGISTER_BACKOFF_MULTIPLIER, attempts) * REGISTER_BACKOFF));
            log.debug(String.format("Waiting for %sms until next registration attempt.", backoff));

            try {
                Thread.sleep(backoff);
            } catch (InterruptedException e) {
                log.info("Shutting down due to interrupt");
                this.shutdown.set(true);
                Thread.currentThread().interrupt();
            }
        }
        if (!this.shutdown.get() && !this.connected.getAndSet(true)) {
            log.debug(String.format("Established connection to %s.", this.registerUrl));
        }
    }

    private HttpResponse registerAgent(URL callbackUrl, byte[] putDataBytes) {
        log.debug("Registering  javaagent on " + callbackUrl + " with " + new String(putDataBytes));
        OutputStream outputStream = null;
        HttpResponse httpResponse = null;
        try {
            HttpURLConnection connection = (HttpURLConnection) callbackUrl.openConnection(Proxy.NO_PROXY);
            connection.setUseCaches(false);
            connection.setConnectTimeout(REGISTER_CONNECT_TIMEOUT);
            connection.setReadTimeout(REGISTER_READ_TIMEOUT);
            connection.setRequestMethod("PUT");
            connection.setRequestProperty("Content-Length", Integer.toString(putDataBytes.length));
            connection.setRequestProperty("Content-Type", "text/plain");
            connection.setDoOutput(true);
            outputStream = connection.getOutputStream();
            outputStream.write(putDataBytes);
            int responseCode = connection.getResponseCode();
            httpResponse = new HttpResponse(responseCode, null);
        } catch (IOException e) {
            log.error("Javaagent could not be registered on " + callbackUrl + ": " + e.getMessage());
        } finally {
            if (outputStream != null) {
                try {
                    outputStream.close();
                } catch (IOException e) {
                    //ignore can't do a thing about this
                }
            }
        }
        return httpResponse;
    }

    public boolean shutdown() {
        if (this.shutdown.getAndSet(true)) {
            return false;
        } else {
            log.debug("Shutting down JavaAgentThread.");
            if (this.connected.getAndSet(false)) {
                this.lostConnection();
            }
            if (this.serverSocket != null) {
                try {
                    this.serverSocket.close();
                } catch (IOException ex) {
                    log.error("Could not shutdown ServerSocket: " + ex.getMessage());
                }
            }
            this.loadAgentPluginCommandHandler.destroy();
            return true;
        }
    }

    private void lostConnection() {
        this.loadedClassesCache.clear();
    }

    private static class HttpResponse {
        private final int code;
        private final String body;

        private HttpResponse(int code, String body) {
            this.code = code;
            this.body = body;
        }
    }

}
