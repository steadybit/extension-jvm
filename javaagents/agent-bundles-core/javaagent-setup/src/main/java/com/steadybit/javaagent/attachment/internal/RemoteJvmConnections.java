/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.net.InetAddress;
import java.net.InetSocketAddress;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;

public class RemoteJvmConnections {
    private static final Logger log = LoggerFactory.getLogger(RemoteJvmConnections.class);
    private final Map<Integer, InetSocketAddress> connections = new ConcurrentHashMap<>();

    public boolean waitForConnection(Integer pid, long timeout) {
        var deadline = System.currentTimeMillis() + timeout;
        try {
            while (System.currentTimeMillis() <= deadline) {
                if (this.connections.containsKey(pid)) {
                    return true;
                }
                Thread.sleep(100L);
            }
        } catch (InterruptedException e) {
            log.warn("Interrupted waiting for RemoteJvmConnection.");
            //interrupt self so that the attachment loop in javadiscovery for multiple JVMs  is stopped
            Thread.currentThread().interrupt();
        }
        return false;
    }

    public InetSocketAddress getConnection(Integer pid) {
        return this.connections.get(pid);
    }

    public void addConnection(Integer pid, InetAddress host, Integer port) {
        var connection = new InetSocketAddress(host, port);

        var existingConnection = this.connections.get(pid);
        if (existingConnection != null && existingConnection.equals(connection)) {
            log.trace("JVM connection with  with PID {} on {} exists already. Skipping registration.", pid, connection);
            return;
        }

        existingConnection = this.connections.put(pid, connection);
        if (existingConnection == null) {
            log.debug("Registered new JVM connection with PID {} on {}", pid, connection);
        } else {
            log.debug("Replaced existing JVM connection with PID {} on {}", pid, connection);
        }
    }

    public void removeConnection(Integer pid) {
        this.connections.remove(pid);
    }

    public Integer size() {
        return this.connections.size();
    }

    public void clear() {
        this.connections.clear();
    }
}
