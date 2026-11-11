/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.instrumentation;

import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;
import net.bytebuddy.agent.builder.AgentBuilder;
import net.bytebuddy.description.type.TypeDescription;
import net.bytebuddy.matcher.ElementMatcher;
import net.bytebuddy.utility.JavaModule;

import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicReference;

/**
 * ByteBuddy swallows errors raised while transforming a class. This listener reports them and keeps the first one, so
 * that callers can fail instead of continuing with an instrumentation that has no effect.
 */
public class TransformationErrorListener extends AgentBuilder.Listener.Adapter {
    private static final String JAVA_VERSION = System.getProperty("java.version");
    private static final Logger log = RemoteAgentLogger.getLogger(TransformationErrorListener.class);
    private final AtomicReference<IllegalStateException> firstError = new AtomicReference<>();
    private final Set<String> targets = ConcurrentHashMap.newKeySet();
    private volatile boolean targetsOnly;

    /**
     * Restricts this listener to errors of the types accepted by the given matcher. ByteBuddy evaluates a type matcher
     * against every class and also reports a failing evaluation, which says nothing about the types to instrument.
     */
    public ElementMatcher<TypeDescription> targets(ElementMatcher<? super TypeDescription> typeMatcher) {
        this.targetsOnly = true;
        return type -> {
            boolean matches = typeMatcher.matches(type);
            if (matches) {
                this.targets.add(type.getName());
            }
            return matches;
        };
    }

    @Override
    public void onError(String typeName, ClassLoader classLoader, JavaModule module, boolean loaded, Throwable throwable) {
        // On a JVM the bytecode library cannot handle, every loaded class fails. Only the first failure is reported to
        // keep the class loading path of the application cheap.
        if (this.firstError.get() != null || (this.targetsOnly && !this.targets.contains(typeName))) {
            return;
        }
        String message = "Failed to instrument " + typeName + " on Java " + JAVA_VERSION;
        if (this.firstError.compareAndSet(null, new IllegalStateException(message + ": " + throwable, throwable))) {
            log.warn(message, throwable);
        }
    }

    /**
     * @param cleanup runs before the first recorded error is thrown
     */
    public void throwOnError(Runnable cleanup) {
        IllegalStateException error = this.firstError.get();
        if (error != null) {
            cleanup.run();
            throw error;
        }
    }
}
