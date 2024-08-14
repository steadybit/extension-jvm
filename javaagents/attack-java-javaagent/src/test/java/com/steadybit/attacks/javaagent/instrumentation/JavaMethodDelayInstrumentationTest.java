/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import net.bytebuddy.agent.ByteBuddyAgent;
import org.json.JSONArray;
import org.json.JSONObject;
import org.junit.jupiter.api.Test;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.lang.instrument.Instrumentation;
import java.util.Collections;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.data.Offset.offset;

class JavaMethodDelayInstrumentationTest {
    private static final Logger log = LoggerFactory.getLogger(JavaMethodDelayInstrumentationTest.class);
    private static final Instrumentation INSTRUMENTATION = ByteBuddyAgent.install();
    private static final TestClass TEST_CLASS = new TestClass();

    @Test
    void should_delay_method_call() {
        JSONObject config = new JSONObject().put("methods", new JSONArray(Collections.singletonList(TestClass.class.getName() + "#run"))).put("delay", "100");
        JavaMethodDelayInstrumentation attack = new JavaMethodDelayInstrumentation(INSTRUMENTATION, config);

        long normalTime = this.measureTime(TEST_CLASS::run);

        attack.install();
        assertThat(this.measureTime(TEST_CLASS::run)).isCloseTo(normalTime + 100L, offset(10L));
        attack.reset();

        assertThat(this.measureTime(TEST_CLASS::run)).isCloseTo(normalTime, offset(10L));
    }

    @Test
    void should_delay_jitter_method_call() {
        JSONObject config = new JSONObject().put("methods", new JSONArray(Collections.singletonList(TestClass.class.getName() + "#run")))
                .put("delay", "100")
                .put("delayJitter", true);
        JavaMethodDelayInstrumentation attack = new JavaMethodDelayInstrumentation(INSTRUMENTATION, config);

        long normalTime = this.measureTime(TEST_CLASS::run);

        attack.install();
        assertThat(this.measureTime(TEST_CLASS::run)).isCloseTo(normalTime + 100L, offset(30L));
        attack.reset();

        assertThat(this.measureTime(TEST_CLASS::run)).isCloseTo(normalTime, offset(10L));
    }

    private long measureTime(Runnable r) {
        int invocations = 5;
        long start = System.currentTimeMillis();
        for (int i = 0; i < invocations; i++) {
            r.run();
        }
        return (System.currentTimeMillis() - start) / invocations;
    }

    public static class TestClass {
        private void run() {
            log.info("run()");
        }
    }
}