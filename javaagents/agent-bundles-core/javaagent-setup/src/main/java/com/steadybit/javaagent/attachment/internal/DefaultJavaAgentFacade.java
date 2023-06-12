/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import com.steadybit.agent.commons.AgentUtils;
import com.steadybit.agent.resources.EmbeddedResourceHelper;
import com.steadybit.javaagent.attachment.CommandResult;
import com.steadybit.javaagent.attachment.JavaAgentFacade;
import com.steadybit.javaagent.attachment.JavaVm;
import com.steadybit.javaagent.attachment.JavaVms;
import com.steadybit.javaagent.attachment.JvmAttachmentException;
import static java.util.Collections.singletonMap;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import org.springframework.core.io.ClassPathResource;
import org.springframework.core.io.Resource;

import javax.annotation.PostConstruct;
import javax.annotation.PreDestroy;
import java.io.BufferedReader;
import java.io.File;
import java.io.IOException;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.io.OutputStreamWriter;
import java.io.PrintWriter;
import java.net.ConnectException;
import java.net.Socket;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;
import java.util.concurrent.ArrayBlockingQueue;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.CopyOnWriteArrayList;
import java.util.concurrent.RejectedExecutionException;
import java.util.concurrent.ThreadFactory;
import java.util.concurrent.ThreadPoolExecutor;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicInteger;
import java.util.function.BiFunction;
import java.util.function.Supplier;

/**
 * The JavaAgentFacade manages the javaagent attachment and it's plugins.
 */
public class DefaultJavaAgentFacade implements JavaAgentFacade {
    public static final Resource JAVAAGENT_INIT_JAR = new ClassPathResource("/javaagent/javaagent-init.jar");
    public static final Resource JAVAAGENT_MAIN_JAR = new ClassPathResource("/javaagent/javaagent-main.jar");
    private static final int SOCKET_TIMEOUT = (int) TimeUnit.SECONDS.toMillis(10L); //should be lower than the discovery timeout (default 20s)
    private static final Logger log = LoggerFactory.getLogger(DefaultJavaAgentFacade.class);
    private static final Logger jvmLog = LoggerFactory.getLogger("com.steadybit.javaagent.JVM");
    private final ThreadPoolExecutor executor = new ThreadPoolExecutor(1, 4, 30L, TimeUnit.SECONDS, new ArrayBlockingQueue<>(128), new WorkerThreadFactory());
    private final List<AutoloadPlugin> autoloadPlugins = new CopyOnWriteArrayList<>();
    private final PluginTracking pluginTracking = new PluginTracking();
    private final JavaProcessWatcher javaProcessWatcher;
    private final RemoteJvmConnections remoteJvmConnections;
    private final JvmAttachmentFactory jvmAttachmentFactory;
    private final JavaVms javaVms;
    private final JavaVms.Listener jvmListener;
    private final EmbeddedResourceHelper embeddedResourceHelper;
    private final Supplier<Integer> agentHttpPort;

    public DefaultJavaAgentFacade(JavaProcessWatcher javaProcessWatcher, RemoteJvmConnections remoteJvmConnections,
            JvmAttachmentFactory jvmAttachmentFactory, JavaVms javaVms, EmbeddedResourceHelper embeddedResourceHelper,
            Supplier<Integer> agentHttpPort) {
        this.javaProcessWatcher = javaProcessWatcher;
        this.remoteJvmConnections = remoteJvmConnections;
        this.jvmAttachmentFactory = jvmAttachmentFactory;
        this.javaVms = javaVms;
        this.embeddedResourceHelper = embeddedResourceHelper;
        this.agentHttpPort = agentHttpPort;
        this.jvmListener = new JavaVms.Listener() {
            @Override
            public void addedJvm(JavaVm jvm) {
                DefaultJavaAgentFacade.this.attach(new AttachJvmWork(jvm));
            }

            @Override
            public void removedJvm(JavaVm jvm) {
                DefaultJavaAgentFacade.this.abortAttach(jvm.getPid());
                DefaultJavaAgentFacade.this.pluginTracking.removeAll(jvm);
            }
        };
    }

    @PostConstruct
    public void start() {
        if ("true".equalsIgnoreCase(AgentUtils.getConfigParam("STEADYBIT_AGENT_JVM_ATTACHMENT_ENABLED", "true"))) {
            this.javaVms.addListener(this.jvmListener);
        }
    }

    @PreDestroy
    public void stop() {
        this.javaVms.removeListener(this.jvmListener);
        this.executor.shutdown();
    }

    private void attach(AttachJvmWork work) {
        try {
            this.executor.submit(work);
        } catch (RejectedExecutionException ex) {
            log.error("Too much pending jvm attachments, skipped {}", work.jvm, ex);
        }
    }

    private void abortAttach(int pid) {
        this.executor.getQueue().removeIf(r -> r instanceof AttachJvmWork && ((AttachJvmWork) r).jvm.getPid() == pid);
    }

    private void doAttach(AttachJvmWork work) {
        var jvm = work.jvm;
        try {
            if (this.attachInternal(jvm)) {
                log.debug("Successful attachment to JVM {}", jvm);

                var loglevel = this.getJvmAgentLogLevel();
                log.trace("Propagating Loglevel {} to Javaagent in JVM {}", loglevel, jvm);
                if (!this.setLogLevel(jvm, loglevel)) {
                    //If setting the loglevel fails, the connection has some issue - do retry
                    this.attach(work);
                }

                for (var plugin : this.autoloadPlugins) {
                    this.loadAutoloadPlugin(jvm, plugin.markerClass, plugin.plugin);
                }
            } else {
                log.debug("Attach to JVM skipped. Excluding {}", jvm);
            }
        } catch (Exception ex) {
            if (this.javaProcessWatcher.isRunning(jvm.getPid())) {
                if (log.isDebugEnabled()) {
                    log.warn("Attach to JVM failed. {} ", jvm.toDebugString(), ex);
                } else {
                    log.warn("Attach to JVM failed. {}: {}", jvm, ex.getMessage());
                }
            } else {
                log.trace("Jvm stopped, attach failed. {} ", jvm.toDebugString(), ex);
            }
        }
    }

    @Override
    public boolean isAttached(JavaVm vm) {
        return this.remoteJvmConnections.getConnection(vm.getPid()) != null;
    }

    /**
     * Attaches the main javaagent to the given VirtualMachine.
     * If there is already an agent attached, attachment will be skipped.
     */
    boolean attachInternal(JavaVm jvm) {
        if (this.isAttached(jvm)) {
            log.trace("RemoteJvmConnection to JVM already established. {}", jvm);
            return true;
        }

        log.debug("RemoteJvmConnection to JVM not found. Attaching now. {}", jvm);

        File mainJar;
        File initJar;
        try {
            mainJar = this.embeddedResourceHelper.toFile(JAVAAGENT_MAIN_JAR);
        } catch (IOException ex) {
            throw new JvmAttachmentException("javaagent-main.jar could not be prepared. JVM attachment is not performed.", ex);
        }
        try {
            initJar = this.embeddedResourceHelper.toFile(JAVAAGENT_INIT_JAR);
        } catch (IOException e) {
            throw new JvmAttachmentException("javaagent-init.jar could not be prepared. JVM attachment is not performed.", e);
        }

        var attached = this.jvmAttachmentFactory.getAttachment(jvm).attach(mainJar, initJar, this.agentHttpPort.get());
        if (!attached) {
            return false;
        }

        var jvmConnectionPresent = this.remoteJvmConnections.waitForConnection(jvm.getPid(), 90_000L);
        if (!jvmConnectionPresent) {
            throw new JvmAttachmentException("JVM with did not call back after 90 seconds.");
        }

        return true;
    }

    @Override
    public void addAutoloadAgentPlugin(Resource plugin, String markerClass) {
        this.autoloadPlugins.add(new AutoloadPlugin(plugin, markerClass));

        for (var jvm : this.javaVms.getJavaVms()) {
            this.loadAutoloadPlugin(jvm, markerClass, plugin);
        }
    }

    @Override
    public void removeAutoloadAgentPlugin(Resource plugin, String markerClass) {
        this.autoloadPlugins.removeIf(p -> p.plugin.equals(plugin) && p.markerClass.equals(markerClass));
        for (var jvm : this.javaVms.getJavaVms()) {
            this.unloadAutoloadPlugin(jvm, markerClass, plugin);
        }
    }

    private void loadAutoloadPlugin(JavaVm jvm, String markerClass, Resource plugin) {
        if (this.hasClassLoaded(jvm, markerClass)) {
            log.debug("Autoloading plugin {} for {}.", plugin, jvm.toDebugString());

            try {
                this.loadAgentPlugin(jvm, plugin, "");
            } catch (Exception ex) {
                log.warn("Autoloading plugin {} for {} failed.", plugin, jvm.toDebugString(), ex);
            }
        }
    }

    private void unloadAutoloadPlugin(JavaVm jvm, String markerClass, Resource plugin) {
        if (this.hasClassLoaded(jvm, markerClass)) {
            log.trace("Unloading plugin {} for {}.", plugin, jvm.toDebugString());

            try {
                this.unloadAgentPlugin(jvm, plugin);
            } catch (Exception ex) {
                log.warn("Unloading plugin {} for {} failed.", plugin, jvm.toDebugString(), ex);
            }
        }
    }

    /**
     * Sets the loglevel for the javaagent attached to the VM with the given PID.
     */
    @Override
    public boolean setLogLevel(JavaVm jvm, String logLevel) {
        return this.sendCommandToAgent(jvm, "log-level", logLevel);
    }

    /**
     * Queries the javaagent with the given pid if a certain class is loaded.
     */
    @Override
    public boolean hasClassLoaded(JavaVm jvm, String className) {
        return this.sendCommandToAgent(jvm, "class-loaded", className);
    }

    @Override
    public boolean hasAgentPlugin(JavaVm jvm, Resource plugin) {
        return this.pluginTracking.has(jvm, plugin);
    }

    /**
     * Loads the given maven artifact as a agent plugin into the agent attached to the VM with the given PID
     */
    @Override
    public boolean loadAgentPlugin(JavaVm jvm, Resource plugin, String args) {
        if (this.hasAgentPlugin(jvm, plugin)) {
            return true;
        }

        File agentPluginJar;
        try {
            agentPluginJar = this.embeddedResourceHelper.toFile(plugin);
        } catch (IOException e) {
            log.error("{} could not be downloaded. JVM attachment is not performed.", plugin, e);
            return false;
        }

        String pluginPath;
        if (jvm.isRunningInContainer()) {
            var destFileName = "steadybit-" + plugin.getFilename();
            this.jvmAttachmentFactory.getAttachment(jvm).copyFiles("/tmp", singletonMap(destFileName, agentPluginJar));
            pluginPath = "/tmp/" + destFileName;
        } else {
            pluginPath = agentPluginJar.toString();
        }

        var loaded = this.sendCommandToAgent(jvm, "load-agent-plugin", pluginPath + "," + args);
        if (loaded) {
            this.pluginTracking.add(jvm, plugin);
        }
        return loaded;
    }

    /**
     * Removes the given maven artifact as a agent plugin from the agent attached to the VM with the given PID
     */
    @Override
    public boolean unloadAgentPlugin(JavaVm jvm, Resource plugin) {
        File agentPluginJar;
        try {
            agentPluginJar = this.embeddedResourceHelper.toFile(plugin);
        } catch (IOException e) {
            log.error("{} could not be downloaded. JVM attachment is not performed.", plugin, e);
            return false;
        }

        var args = agentPluginJar.toString();
        if (jvm.isRunningInContainer()) {
            var destFileName = "steadybit-" + plugin.getFilename();
            args = "/tmp/" + destFileName + ",deleteFile=true";
        }

        var unloaded = this.sendCommandToAgent(jvm, "unload-agent-plugin", args);
        if (unloaded) {
            this.pluginTracking.remove(jvm, plugin);
        }
        return unloaded;
    }

    @Override
    public boolean sendCommandToAgent(JavaVm jvm, String command, String args) {
        var success = this.sendCommandToAgent(jvm, command, args, (inputStream, result) -> {
            try {
                var br = new BufferedReader(new InputStreamReader(inputStream, StandardCharsets.UTF_8));
                var response = br.readLine();
                if (CommandResult.OK.equals(result)) {
                    log.trace("Command '{}:{}' to agent on PID {} returned: {}", command, args, jvm.getPid(), response);
                    return response != null && response.equals("true");
                } else {
                    log.warn("Command '{}:{}' to agent on PID {} returned error: {}", command, args, jvm.getPid(), response);
                    return false;
                }
            } catch (IOException ex) {
                log.warn("Command '{}:{}' to agent on PID {} threw exception", command, args, jvm.getPid(), ex);
                return false;
            }
        });
        return success != null && success;
    }

    @Override
    public <T> T sendCommandToAgent(JavaVm jvm, String command, String args, BiFunction<InputStream, CommandResult, T> handler) {
        var pid = jvm.getPid();
        var inetSocketAddress = this.remoteJvmConnections.getConnection(pid);
        if (inetSocketAddress == null) {
            log.debug("RemoteJvmConnection from PID {} not found. Command '{}:{}' not sent.", pid, command, args);
            return null;
        }

        args = args != null ? args : "";

        try (var socket = new Socket()) {
            socket.connect(inetSocketAddress, SOCKET_TIMEOUT);
            socket.setSoTimeout(SOCKET_TIMEOUT);
            log.trace("Sending command '{}:{}' to agent on PID {}", command, args, pid);
            var printWriter = new PrintWriter(new OutputStreamWriter(socket.getOutputStream(), StandardCharsets.UTF_8), true);
            printWriter.println(command + ':' + args);
            var response = socket.getInputStream();
            return handler.apply(response, CommandResult.of(response.read()));
        } catch (IOException e) {
            if (this.javaProcessWatcher.isRunning(pid)) {
                if (e instanceof ConnectException) {
                    log.error("Command '{}' could not be sent over socket to {} ({}): {}", command, jvm, inetSocketAddress, e.getMessage());
                } else {
                    log.error("Command '{}' could not be sent over socket to {} ({}):", command, jvm, inetSocketAddress, e);
                }
            } else if (!Thread.currentThread().isInterrupted()) {
                //if the current thread is interrupted oshi will report null for the process which is incorrect.
                //when the discovery is running into a timeout, this is most of the case.
                //so we only remove the process if we weren't interrupted
                if (log.isDebugEnabled()) {
                    log.debug("Process is not running anymore. Removing connection to {}: {}", jvm, inetSocketAddress);
                }
                this.remoteJvmConnections.removeConnection(pid);
            }
            return null;
        }
    }

    @Override
    public String getAgentHost(JavaVm jvm) {
        return this.jvmAttachmentFactory.getAttachment(jvm).getAgentHost();
    }

    private String getJvmAgentLogLevel() {
        if (jvmLog.isTraceEnabled()) {
            return "TRACE";
        } else if (jvmLog.isDebugEnabled()) {
            return "DEBUG";
        } else if (jvmLog.isInfoEnabled()) {
            return "INFO";
        } else if (jvmLog.isWarnEnabled()) {
            return "WARN";
        } else {
            return jvmLog.isErrorEnabled() ? "ERROR" : "OFF";
        }
    }

    private static class AutoloadPlugin {
        private final Resource plugin;
        private final String markerClass;

        private AutoloadPlugin(Resource resource, String markerClass) {
            this.plugin = resource;
            this.markerClass = markerClass;
        }
    }

    class AttachJvmWork implements Runnable {
        private final JavaVm jvm;
        private int retries = 5;

        AttachJvmWork(JavaVm jvm) {
            this.jvm = jvm;
        }

        @Override
        public void run() {
            if (this.retries-- > 0) {
                DefaultJavaAgentFacade.this.doAttach(this);
            } else {
                log.warn("Attach retries for {} exceeded.", this.jvm);
            }
        }
    }

    private static class WorkerThreadFactory implements ThreadFactory {
        private final AtomicInteger threadNumber = new AtomicInteger(1);
        private final ThreadGroup group = Thread.currentThread().getThreadGroup();

        @Override
        public Thread newThread(Runnable runnable) {
            return new Thread(this.group, runnable, "jvm-attach-" + this.threadNumber.getAndIncrement());
        }
    }

    private static class PluginTracking {
        private final Map<Integer, List<Resource>> plugins = new ConcurrentHashMap<>();

        void add(JavaVm jvm, Resource plugin) {
            this.plugins.compute(jvm.getPid(), (key, oldValue) -> {
                if (oldValue == null) {
                    oldValue = new ArrayList<>();
                }
                oldValue.add(plugin);
                return oldValue;
            });
        }

        void remove(JavaVm jvm, Resource plugin) {
            this.plugins.compute(jvm.getPid(), (key, oldValue) -> {
                if (oldValue == null) {
                    return oldValue;
                }
                oldValue.remove(plugin);
                return oldValue;
            });
        }

        void removeAll(JavaVm jvm) {
            this.plugins.remove(jvm.getPid());
        }

        boolean has(JavaVm jvm, Resource plugin) {
            var list = this.plugins.get(jvm.getPid());
            return list != null && list.contains(plugin);
        }
    }
}
