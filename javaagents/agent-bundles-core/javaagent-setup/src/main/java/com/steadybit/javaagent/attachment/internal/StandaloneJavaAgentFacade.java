/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import com.steadybit.agent.resources.EmbeddedResourceHelper;
import com.steadybit.cri.CriClient;
import com.steadybit.docker.DockerClient;
import com.steadybit.javaagent.attachment.CommandResult;
import com.steadybit.javaagent.attachment.JavaAgentFacade;
import com.steadybit.javaagent.attachment.JavaVm;
import com.sun.net.httpserver.HttpServer;
import org.apache.commons.io.IOUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.core.io.Resource;

import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.ScheduledThreadPoolExecutor;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.function.BiFunction;

/**
 * A Javaagent-Agent loader with all the necessary dependencies. For test use only.
 */
public class StandaloneJavaAgentFacade implements JavaAgentFacade {
    private final DockerClient dockerClient = new DockerClient();
    private final ScheduledThreadPoolExecutor executor = new ScheduledThreadPoolExecutor(2);
    private final DefaultJavaAgentFacade delegate;
    private final ProxyingRemoteJvmConnections remoteJvmConnections;
    private final DefaultJavaVms javaVms;
    private final DockerDesktopAwareJavaProcessWatcher javaProcessWatcher;
    private final HotspotJvmWatcher hotspotJvmWatcher;
    private final RegisterJvmConnectionsHttpEndpoint httpEndpoint;
    private final OverridingEmbeddedResourceHelper embeddedResourceHelper;

    public StandaloneJavaAgentFacade() {
        this.remoteJvmConnections = new ProxyingRemoteJvmConnections();
        this.httpEndpoint = new RegisterJvmConnectionsHttpEndpoint(42899, this.executor, this.remoteJvmConnections);
        this.javaProcessWatcher = new DockerDesktopAwareJavaProcessWatcher(this.dockerClient);
        this.hotspotJvmWatcher = new HotspotJvmWatcher();
        this.javaVms = new DefaultJavaVms(this.hotspotJvmWatcher, this.javaProcessWatcher, this.dockerClient, new CriClient());
        this.embeddedResourceHelper = new OverridingEmbeddedResourceHelper();
        var jvmAttachmentFactory = new JvmAttachmentFactory(this.dockerClient, new CriClient(), this.javaProcessWatcher);

        this.delegate = new DefaultJavaAgentFacade(this.javaProcessWatcher, this.remoteJvmConnections, jvmAttachmentFactory, this.javaVms,
                this.embeddedResourceHelper, this.httpEndpoint::getPort);
    }

    public void addJvm(JavaVm jvm) {
        this.javaVms.addJvm(jvm);
        if (jvm.isRunningInContainer()) {
            this.javaProcessWatcher.addContainerProcess(jvm);
        }
    }

    public void removeJvm(JavaVm jvm) {
        this.javaVms.removeJvm(jvm.getPid());
    }

    public void waitForAttachment(JavaVm jvm) {
        this.remoteJvmConnections.waitForConnection(jvm.getPid(), 30_000L);
    }

    public void start() {
        this.dockerClient.activate();
        this.executor.scheduleWithFixedDelay(this.javaProcessWatcher::updatePids, 0, 1, java.util.concurrent.TimeUnit.SECONDS);
        this.executor.scheduleWithFixedDelay(this.hotspotJvmWatcher::updatePids, 0, 1, java.util.concurrent.TimeUnit.SECONDS);
        this.javaVms.activate();
        this.httpEndpoint.start();
        this.delegate.start();
    }

    public void stop() {
        this.executor.shutdown();
        this.httpEndpoint.stop();
        this.delegate.stop();
        this.javaVms.deactivate();
    }

    @Override
    public boolean isAttached(JavaVm vm) {
        return this.delegate.isAttached(vm);
    }

    public boolean attachInternal(JavaVm jvm) {
        return this.delegate.attachInternal(jvm);
    }

    @Override
    public void addAutoloadAgentPlugin(Resource plugin, String markerClass) {
        this.delegate.addAutoloadAgentPlugin(plugin, markerClass);
    }

    @Override
    public void removeAutoloadAgentPlugin(Resource plugin, String markerClass) {
        this.delegate.removeAutoloadAgentPlugin(plugin, markerClass);
    }

    @Override
    public boolean setLogLevel(JavaVm jvm, String logLevel) {
        return this.delegate.setLogLevel(jvm, logLevel);
    }

    @Override
    public boolean hasClassLoaded(JavaVm jvm, String className) {
        return this.delegate.hasClassLoaded(jvm, className);
    }

    @Override
    public boolean hasAgentPlugin(JavaVm jvm, Resource plugin) {
        return this.delegate.hasAgentPlugin(jvm, plugin);
    }

    @Override
    public boolean loadAgentPlugin(JavaVm jvm, Resource plugin, String args) {
        return this.delegate.loadAgentPlugin(jvm, plugin, args);
    }

    @Override
    public boolean unloadAgentPlugin(JavaVm jvm, Resource plugin) {
        return this.delegate.unloadAgentPlugin(jvm, plugin);
    }

    @Override
    public boolean sendCommandToAgent(JavaVm jvm, String command, String args) {
        return this.delegate.sendCommandToAgent(jvm, command, args);
    }

    @Override
    public <T> T sendCommandToAgent(JavaVm jvm, String command, String args,
            BiFunction<InputStream, CommandResult, T> handler) {
        return this.delegate.sendCommandToAgent(jvm, command, args, handler);
    }

    @Override
    public String getAgentHost(JavaVm jvm) {
        return this.delegate.getAgentHost(jvm);
    }

    public BaseJavaVms getJavaVms() {
        return this.javaVms;
    }

    public RemoteJvmConnections getRemoteJvmConnections() {
        return this.remoteJvmConnections;
    }

    public void setProxyRemoteJvmConnection(BiFunction<Integer, InetSocketAddress, InetSocketAddress> proxyFn) {
        this.remoteJvmConnections.setProxyFunction(proxyFn);
    }

    public void addResourceOverride(Resource resource, Resource override) {
        this.embeddedResourceHelper.addOverride(resource, override);
    }

    private static class ProxyingRemoteJvmConnections extends RemoteJvmConnections {
        private final Map<Integer, InetSocketAddress> startedProxies = new ConcurrentHashMap<>();
        private BiFunction<Integer, InetSocketAddress, InetSocketAddress> proxyFunction = (pid, address) -> address;

        @Override
        public synchronized InetSocketAddress getConnection(Integer pid) {
            var connection = super.getConnection(pid);
            if (connection != null) {
                return this.startedProxies.computeIfAbsent(pid, (p) -> this.proxyFunction.apply(p, connection));
            }
            return connection;
        }

        public void setProxyFunction(BiFunction<Integer, InetSocketAddress, InetSocketAddress> proxyFunction) {
            this.proxyFunction = proxyFunction;
        }
    }

    private static class DockerDesktopAwareJavaProcessWatcher extends JavaProcessWatcher {
        private final Map<Integer, String> containerProcesses = new HashMap<>();
        private final DockerClient dockerClient;

        private DockerDesktopAwareJavaProcessWatcher(DockerClient dockerClient) {
            this.dockerClient = dockerClient;
        }

        public void addContainerProcess(JavaVm jvm) {
            this.containerProcesses.put(jvm.getPid(), jvm.getContainerId());
        }

        @Override
        public boolean isRunning(int pid) {
            var containerId = this.containerProcesses.get(pid);
            if (containerId != null) {
                if (this.dockerClient.isRunning(containerId)) {
                    return true;
                }
                this.containerProcesses.remove(pid);
            }

            return super.isRunning(pid);
        }
    }

    private static class RegisterJvmConnectionsHttpEndpoint {
        private static final Logger log = LoggerFactory.getLogger(RegisterJvmConnectionsHttpEndpoint.class);
        private final HttpServer server;
        private final RegisterJavaAgentHandler handler;

        private RegisterJvmConnectionsHttpEndpoint(int port, ThreadPoolExecutor executor, RemoteJvmConnections remoteJvmConnections) {
            this.handler = new RegisterJavaAgentHandler(remoteJvmConnections);
            try {
                this.server = HttpServer.create();
                this.server.setExecutor(executor);
                this.server.createContext("/javaagent", exchange -> {
                    var body = IOUtils.toString(exchange.getRequestBody(), StandardCharsets.UTF_8);
                    var status = this.handler.handleInternal(exchange.getRemoteAddress().getAddress(), body);
                    exchange.sendResponseHeaders(status, 0);
                });
            } catch (IOException e) {
                throw new RuntimeException(e);
            }
        }

        private void start() {
            log.trace("Starting AgentHttpEndpoint on port {}...", 0);
            try {
                this.server.bind(new InetSocketAddress(0), 0);
                this.server.start();
                log.info("AgentHttpEndpoint available on http://{}/*", RegisterJvmConnectionsHttpEndpoint.this.server.getAddress());
            } catch (IOException e) {
                throw new RuntimeException("Could not start HttpEndpoint", e);
            }
        }

        private void stop() {
            this.server.stop(5);
        }

        public int getPort() {
            if (this.server.getAddress() == null) {
                throw new IllegalStateException("server is not started");
            }
            return this.server.getAddress().getPort();
        }
    }

    private static class OverridingEmbeddedResourceHelper extends EmbeddedResourceHelper {
        private final Map<Resource, Resource> overrides = new ConcurrentHashMap<>();

        @Override
        public File toFile(Resource resource) throws IOException {
            return super.toFile(this.overrides.getOrDefault(resource, resource));
        }

        private void addOverride(Resource resource, Resource override) {
            this.overrides.put(resource, override);
        }
    }
}
