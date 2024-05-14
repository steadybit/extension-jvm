/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.springboot2;

import com.steadybit.javaagent.AgentPlugin;

import java.lang.instrument.Instrumentation;

/**
 * AgentPlugin to execute attacks inside a JVM
 */
public class AttackSpringboot2AgentPlugin implements AgentPlugin {
    private final String attackUrl;
    private final Instrumentation instrumentation;
    private Thread thread;

    public AttackSpringboot2AgentPlugin(String attackUrl, Instrumentation instrumentation) {
        this.attackUrl = attackUrl;
        this.instrumentation = instrumentation;
    }

    @Override
    public void start() {
        this.thread = new Thread(new AttackRunnable(this.instrumentation, this.attackUrl));
        this.thread.start();
    }

    @Override
    public void destroy() {
        if (this.thread != null) {
            this.thread.interrupt();
            this.thread = null;
        }
    }
}
