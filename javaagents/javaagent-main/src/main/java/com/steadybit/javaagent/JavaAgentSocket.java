/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent;

import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;

import java.io.IOException;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.MalformedURLException;
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
public class JavaAgentSocket extends Thread {
  private static final Logger log = RemoteAgentLogger.getLogger(JavaAgentSocket.class);
  private final int acceptTimeout = (int) TimeUnit.SECONDS.toMillis(60L);
  private static final int REGISTER_CONNECT_TIMEOUT = 2000;
  private static final int REGISTER_READ_TIMEOUT = 5_000;
  private static final long REGISTER_BACKOFF = 1000L;
  private static final long REGISTER_BACKOFF_MAX = 15_000L;
  private static final long REGISTER_BACKOFF_MULTIPLIER = 2L;
  private final String pid;
  private final String agentHost;
  private final AtomicBoolean connected = new AtomicBoolean(false);
  private final AtomicBoolean shutdown = new AtomicBoolean(false);
  private final URL registerUrl;
  private final JavaAgentSocketHandler socketHandler;
  private ServerSocket serverSocket;
  private String remoteAddress;

  public JavaAgentSocket(String pid, String agentHost, String agentPort, JavaAgentSocketHandler acceptHandler) throws MalformedURLException {
    super("steadybit-javaagent");
    this.pid = pid;
    this.agentHost = agentHost;
    this.registerUrl = new URL(String.format("http://%s:%s/javaagent", this.agentHost, agentPort));
    this.setUncaughtExceptionHandler((t, e) -> log.error("Uncaught exception", e));
    this.socketHandler = acceptHandler;
  }

  @Override
  public void run() {
    try {
      this.init();
      if (this.announce()) {
        this.listen();
      }
    } catch (Exception e) {
      log.error("Could not init and listen", e);
    } finally {
      this.shutdown();
    }
  }

  private void init() throws IOException {
    this.serverSocket = new ServerSocket(0, 0);
    if ("127.0.0.1".equals(this.agentHost)) {
      this.remoteAddress = String.format("%s=%s:%s", this.pid, "127.0.0.1", this.serverSocket.getLocalPort());
    } else {
      this.remoteAddress = String.format("%s=%s", this.pid, this.serverSocket.getLocalPort());
    }
    log.debug(String.format("Created ServerSocket with remote address %s", this.remoteAddress));
    this.serverSocket.setSoTimeout(this.acceptTimeout);
  }

  private void listen() {
    do {
      try {
        this.accept();
      } catch (SocketTimeoutException ex) {
        log.trace(String.format("Could not accept remote connection: %s", ex.getMessage()));
      } catch (IOException ex) {
        if (!this.shutdown.get()) {
          log.debug(String.format("Could not accept remote connection: %s", ex.getMessage()));
        }
      } catch (Exception ex) {
        log.debug("Exception while listening for commands: ", ex);
      }
    } while (!this.shutdown.get());
  }

  private void accept() throws IOException {
    Socket socket = null;
    try {
      log.trace(
        String.format("Accepting data for JVM with PID %s on %s:%s", this.pid,
          this.serverSocket.getInetAddress().getHostAddress(),
          this.serverSocket.getLocalPort())
      );
      socket = this.serverSocket.accept();
      this.socketHandler.accept(socket);
    } finally {
      if (socket != null) {
        try {
          socket.close();
        } catch (IOException e) {
          log.warn("Could not close Socket:" + e.getMessage());
        }
      }
    }
  }

  private boolean announce() {
    int attempts = 0;

    while (!this.shutdown.get()) {
      Integer responseCode = this.registerAgent();
      if (responseCode != null && responseCode >= 200 && responseCode <= 299) {
        log.debug(String.format("Javaagent successfully registered to %s with HTTP Code %s", this.registerUrl, responseCode));
        if (!this.shutdown.get() && !this.connected.getAndSet(true)) {
          log.debug(String.format("Established connection to %s.", this.registerUrl));
        }
        return true;
      }

      if (attempts++ > 60 && this.connected.getAndSet(false)) {
        log.error(String.format("Lost connection to %s", this.registerUrl));
        this.socketHandler.disconnected();
      } else {
        log.debug(String.format("Javaagent failed register with HTTP Code %s", responseCode));
      }

      long backoff = Math.min(REGISTER_BACKOFF_MAX, (long) (Math.pow(REGISTER_BACKOFF_MULTIPLIER, attempts) * REGISTER_BACKOFF));
      log.debug(String.format("Waiting for %sms until next registration attempt.", backoff));

      try {
        Thread.sleep(backoff);
      } catch (InterruptedException e) {
        Thread.currentThread().interrupt();
        return false;
      }
    }
    return false;
  }

  private Integer registerAgent() {
    log.debug(String.format("Registering javaagent on %s with %s", this.registerUrl, this.remoteAddress));

    OutputStream outputStream = null;
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
      outputStream = connection.getOutputStream();
      outputStream.write(content);
      return connection.getResponseCode();
    } catch (IOException e) {
      log.error(String.format("Javaagent could not be registered on %s: %s", this.registerUrl, e.getMessage()), e);
    } finally {
      if (outputStream != null) {
        try {
          outputStream.close();
        } catch (IOException e) {
          log.debug(String.format("Could not close OutputStream: %s", e.getMessage()));
        }
      }
    }
    return null;
  }

  public boolean shutdown() {
    if (this.shutdown.getAndSet(true)) {
      return false;
    }

    log.debug("Shutting down JavaAgentSocket.");
    if (this.connected.getAndSet(false)) {
      this.socketHandler.disconnected();
    }

    if (this.serverSocket != null) {
      try {
        this.serverSocket.close();
      } catch (IOException ex) {
        log.error("Could not shutdown ServerSocket: " + ex.getMessage());
      }
    }
    return true;
  }
}
