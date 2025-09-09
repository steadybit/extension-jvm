/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent;

import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;

import java.io.File;
import java.io.IOException;
import java.io.InterruptedIOException;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.Proxy;
import java.net.ServerSocket;
import java.net.Socket;
import java.net.SocketTimeoutException;
import java.net.URL;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicBoolean;

/**
 * Main Javaagent Thread.
 * Announces itself at the steadybit agent using http and listens on tcp for incoming commands.
 * New connections are delegated to the acceptHandler.
 */
public class JavaAgentSocket {
    private static final Logger log = RemoteAgentLogger.getLogger(JavaAgentSocket.class);
    private static final int ACCEPT_TIMEOUT = (int) TimeUnit.SECONDS.toMillis(30L);
    private static final int REGISTER_CONNECT_TIMEOUT = 2000;
    private static final int REGISTER_READ_TIMEOUT = 5_000;
    private static final long REGISTER_BACKOFF = 1000L;
    private static final long REGISTER_BACKOFF_MULTIPLIER = 3L;
    private static final long REGISTER_BACKOFF_MAX = REGISTER_BACKOFF_MULTIPLIER * 3L * REGISTER_BACKOFF;
    private static final long HEARTBEAT_MAX_AGE = 30_000;
    private final String pid;
    private final AtomicBoolean connected = new AtomicBoolean(false);
    private final AtomicBoolean shutdown = new AtomicBoolean(false);
    private final URL registerUrl;
    private final JavaAgentSocketHandler socketHandler;
    private final File heartbeat;
    private ServerSocket serverSocket;
    private String remoteAddress;
    private final Thread thread;

    public JavaAgentSocket(String pid, URL registerUrl, File heartbeat, JavaAgentSocketHandler acceptHandler) {
        this.pid = pid;
        this.registerUrl = registerUrl;
        this.socketHandler = acceptHandler;
        this.thread = new Thread(this::run, "steadybit-javaagent");
        this.thread.setUncaughtExceptionHandler((t, e) -> log.error("Uncaught exception", e));
        this.heartbeat = heartbeat;
    }

    public void start() {
        this.thread.start();
    }

    private void run() {
        try {
            this.init();

            int attempt = 0;
            while (!(this.shutdown.get() || this.heartbeatTimeoutReached())) {

                if (this.announce()) {
                    attempt = 0;

                    while (!(this.shutdown.get() || this.heartbeatTimeoutReached())) {
                        this.accept();
                    }
                } else {
                    backoff(attempt++);
                }
            }
        } catch (InterruptedException | InterruptedIOException e) {
            Thread.currentThread().interrupt();
            log.trace("Interrupted - shutting down %s", this.toString());
        } catch (Exception e) {
            if (!this.shutdown.get()) {
                log.error("Could not init and listen", e);
            }
        } finally {
            this.shutdown(false);
        }
    }

    private void init() throws IOException {
        this.serverSocket = new ServerSocket(0, 0);
        this.serverSocket.setSoTimeout(ACCEPT_TIMEOUT);
        if ("127.0.0.1".equals(this.registerUrl.getHost())) {
            this.remoteAddress = String.format("%s=%s:%s", this.pid, "127.0.0.1", this.serverSocket.getLocalPort());
        } else {
            this.remoteAddress = String.format("%s=%s", this.pid, this.serverSocket.getLocalPort());
        }
        log.debug("Created ServerSocket with remote address %s", this.remoteAddress);
    }

    private boolean announce() {
        Integer responseCode = this.registerAgent();
        if (responseCode != null && responseCode >= 200 && responseCode <= 299) {
            if (!this.shutdown.get() && !this.connected.getAndSet(true)) {
                RemoteAgentLogger.setConnectedToRemote(true);
                log.debug("Javaagent successfully registered to %s with HTTP Code %s", this.registerUrl, responseCode);
            }
            return true;
        }

        if (this.connected.getAndSet(false)) {
            RemoteAgentLogger.setConnectedToRemote(false);
            log.error("Lost connection to %s", this.registerUrl);
            this.socketHandler.disconnected();
        } else {
            log.debug("Javaagent failed register with HTTP Code %s", responseCode);
        }
        return false;
    }

    private void accept() {
        try (Socket socket = this.serverSocket.accept()) {
            log.trace(
                    "Accepting data for JVM with PID %s on %s:%s", this.pid,
                    this.serverSocket.getInetAddress().getHostAddress(),
                    this.serverSocket.getLocalPort()
            );
            this.socketHandler.accept(socket);
        } catch (SocketTimeoutException ex) {
            log.trace(String.format("Exception while listening for commands: %s", ex.getMessage()));
        } catch (InterruptedIOException ex) {
            log.trace(String.format("Exception while listening for commands: %s", ex.getMessage()));
            Thread.currentThread().interrupt();
        } catch (IOException ex) {
            if (!this.shutdown.get()) {
                log.warn("Exception while listening for commands: ", ex);
            }
        }
    }

    private Integer registerAgent() {
        log.debug("Registering javaagent on %s with %s", this.registerUrl, this.remoteAddress);

        try {
            byte[] content = this.remoteAddress.getBytes(StandardCharsets.UTF_8);
            HttpURLConnection connection = (HttpURLConnection) this.registerUrl.openConnection(Proxy.NO_PROXY);
            connection.setUseCaches(false);
            connection.setConnectTimeout(REGISTER_CONNECT_TIMEOUT);
            connection.setReadTimeout(REGISTER_READ_TIMEOUT);
            connection.setRequestMethod("PUT");
            connection.setRequestProperty("Content-Length", Integer.toString(content.length));
            connection.setRequestProperty("Content-Type", "text/plain");
            connection.setDoOutput(true);
            try (OutputStream outputStream = connection.getOutputStream()) {
                outputStream.write(content);
            }
            return connection.getResponseCode();
        } catch (IOException e) {
            log.error(String.format("Javaagent could not be registered on %s: %s", this.registerUrl, e.getMessage()), e);
        }
        return null;
    }

    private static void backoff(int attempts) throws InterruptedException {
        long backoff = Math.min(REGISTER_BACKOFF_MAX, (long) (Math.pow(REGISTER_BACKOFF_MULTIPLIER, attempts) * REGISTER_BACKOFF));
        log.debug("Waiting for %sms until next registration attempt.", backoff);

        try {
            Thread.sleep(backoff);
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            throw e;
        }
    }

    public void shutdown(boolean wait) {
        if (!this.shutdown.getAndSet(true)) {
            log.debug("Shutting down %s.", this.toString());

            if (this.connected.getAndSet(false)) {
                RemoteAgentLogger.setConnectedToRemote(false);
                this.socketHandler.disconnected();
            }

            if (this.serverSocket != null) {
                try {
                    this.serverSocket.close();
                } catch (IOException ex) {
                    log.error("Could not shutdown ServerSocket: " + ex.getMessage());
                }
            }
        }

        if (wait && this.thread.isAlive()) {
            this.thread.interrupt();
            try {
                this.thread.join(5000L);
            } catch (InterruptedException ex) {
                Thread.currentThread().interrupt();
            }
            if (this.thread.isAlive()) {
                log.error("Failed shutdown JavaAgent - still alive");
            }
        }
    }

    private boolean heartbeatTimeoutReached() {
        if (this.heartbeat == null) {
            return false;
        }

        long lastModified = this.heartbeat.lastModified();
        if (lastModified == 0) {
            log.warn("Heartbeat file '%s' does not exist, shutting down.", this.heartbeat.getPath());
            return true;
        }

        //in case the jvm process has a different clock then the extension
        //we substract the first age.
        long age = System.currentTimeMillis() - lastModified;
        if (age > HEARTBEAT_MAX_AGE) {
            log.warn("Heartbeat file '%s' is too old (%dms), shutting down.", this.heartbeat.getPath(), age);
            return true;
        }
        return false;
    }

    @Override
    public String toString() {
        return JavaAgentSocket.class.getSimpleName() + '[' + this.registerUrl + ']';
    }
}
