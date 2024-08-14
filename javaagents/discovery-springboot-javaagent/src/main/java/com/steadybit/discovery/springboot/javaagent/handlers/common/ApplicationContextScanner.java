/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.common;

import com.steadybit.discovery.springboot.javaagent.instrumentation.ClassTransformationPlugin;
import com.steadybit.javaagent.instrumentation.Registration;
import com.steadybit.javaagent.util.WeakConcurrentSet;
import com.steadybit.shaded.net.bytebuddy.agent.builder.AgentBuilder;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers;
import org.springframework.context.ApplicationContext;

import java.lang.instrument.Instrumentation;
import java.util.ArrayList;
import java.util.List;

import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.hasMethodName;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.named;

public final class ApplicationContextScanner extends ClassTransformationPlugin {
    private final static int STALE_TIME = 2 * 60 * 1000;
    private final WeakConcurrentSet<ApplicationContext> applicationContexts = new WeakConcurrentSet<>();
    private volatile long lastUpdate = 0L;
    private Thread resetThread;

    public ApplicationContextScanner(Instrumentation instrumentation) {
        super(instrumentation);
    }

    @Override
    public void install() {
        super.install();
        this.startRemoveInstrumentationThread();
    }

    @Override
    public void reset() {
        this.resetThread.interrupt();
    }

    @Override
    protected AgentBuilder doInstall(AgentBuilder agentBuilder) {
        return agentBuilder
                .type(named("org.springframework.context.support.AbstractApplicationContext"))
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping().bind(Registration.class, this.getRegistration())).include(
                                CaptureApplicationContextAdvice.class.getClassLoader())
                        .advice(hasMethodName("publishEvent").and(ElementMatchers.takesArguments(1)), CaptureApplicationContextAdvice.class.getName()));
    }

    @Override
    public Object exec(int code, Object arg1) {
        if (code == 0) {
            this.applicationContexts.add((ApplicationContext) arg1);
            this.lastUpdate = System.currentTimeMillis();
        }
        return null;
    }

    private void startRemoveInstrumentationThread() {
        this.resetThread = new Thread(() -> {
            long lastUpdate = this.lastUpdate;
            try {
                while (lastUpdate == 0L || System.currentTimeMillis() < lastUpdate + STALE_TIME) {
                    Thread.sleep(STALE_TIME);
                }
            } catch (InterruptedException e) {
                Thread.currentThread().interrupt();
            } finally {
                super.reset();
            }
        });
        this.resetThread.setName("application-context-scanner-remover");
    }

    public List<ApplicationContext> getApplicationContexts() {
        this.applicationContexts.expungeStaleEntries();
        ArrayList<ApplicationContext> result = new ArrayList<>();
        this.applicationContexts.iterator().forEachRemaining(result::add);
        return result;
    }
}

