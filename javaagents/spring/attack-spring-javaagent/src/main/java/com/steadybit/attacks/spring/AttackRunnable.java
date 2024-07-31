/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.spring;

import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;
import org.json.JSONObject;

import java.lang.reflect.InvocationTargetException;

/**
 * Exectues the Attack inside the JVM.
 * The instrumentation itself is delegated to a InstrumentationAttack.
 * This class manages the lifecycle and the attack status communication with the steadybit-agent using http.
 */
public class AttackRunnable implements Runnable {
    private final HttpAttackClient client;
    private final java.lang.instrument.Instrumentation instrumentation;
    private static final Logger log = RemoteAgentLogger.getLogger(AttackRunnable.class);

    public AttackRunnable(java.lang.instrument.Instrumentation instrumentation, String attackUrl) {
        this.instrumentation = instrumentation;
        this.client = new HttpAttackClient(attackUrl);
    }

    @Override
    public void run() {
        JSONObject config = this.client.fetchAttackConfig();
        log.trace("Running Attack {}", config.toString());
        String attackClass = config.getString("attack-class");
        long duration = config.optLong("duration", 0L);

        try {
            Installable attack = this.createAttack(attackClass, config);
            attack.install();
            log.debug("Bytecode instrumentation for attack {} installed.", config.toString());

            try {
                this.client.attackStarted();
                this.waitForEnd(duration);
            } finally {
                attack.reset();
                log.debug("Bytecode instrumentation for attack {} removed.", config.toString());
                try {
                    this.client.attackStopped();
                } catch (HttpAttackClientException e) {
                    //If the attack was canceled the attack http endpoint is gone and there is no one to report to.
                    log.trace("Could not send attack stop to agent.", e);
                }
            }

        } catch (Exception e) {
            log.warn("Bytecode instrumentation for attack " + config.toString() + " failed.", e);

            try {
                this.client.attackFailed(e);
            } catch (HttpAttackClientException ex) {
                log.trace("Could not send attack failure to agent.", ex);
            }
        }
    }

    private void waitForEnd(long duration) {
        long deadline = System.currentTimeMillis() + duration;
        try {
            while (System.currentTimeMillis() < deadline && this.client.checkAttackStillRunning()) {
                Thread.sleep(500);
            }
        } catch (InterruptedException e) {
            Thread.currentThread().interrupt();
            log.trace("Interrupted waiting on attack end.");
        }
    }

    private Installable createAttack(String attackClass, JSONObject attackConfig) {
        try {
            Class<?> clazz = Class.forName(attackClass);
            Object o = clazz.getConstructor(java.lang.instrument.Instrumentation.class, JSONObject.class).newInstance(this.instrumentation, attackConfig);
            if (o instanceof Installable) {
                return (Installable) o;
            } else {
                throw new IllegalArgumentException("Class '" + attackClass + "' does not implement " + Installable.class);
            }
        } catch (ClassNotFoundException | NoSuchMethodException | InstantiationException | IllegalAccessException | InvocationTargetException e) {
            throw new IllegalArgumentException("Could not instantiate JavaAttack '" + attackClass + "'", e);
        }
    }
}
