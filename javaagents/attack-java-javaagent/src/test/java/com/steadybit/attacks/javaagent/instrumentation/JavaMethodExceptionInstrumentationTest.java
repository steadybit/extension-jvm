/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import net.bytebuddy.agent.ByteBuddyAgent;
import org.json.JSONArray;
import org.json.JSONObject;
import org.junit.jupiter.api.Test;

import java.lang.instrument.Instrumentation;
import java.util.Collections;

import static org.assertj.core.api.Assertions.assertThatCode;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

class JavaMethodExceptionInstrumentationTest {
    private static final Instrumentation INSTRUMENTATION = ByteBuddyAgent.install();
    private static final TestClass TEST = new TestClass();

    @Test
    void should_throw_exception_method_call() {
        JSONObject config = new JSONObject().put("methods", new JSONArray(Collections.singletonList(TestClass.class.getName() + "#run")));
        JavaMethodExceptionInstrumentation attack = new JavaMethodExceptionInstrumentation(INSTRUMENTATION, config);

        assertThatCode(TEST::run).doesNotThrowAnyException();
        attack.install();
        assertThatThrownBy(TEST::run).isInstanceOf(RuntimeException.class);
        attack.reset();
        assertThatCode(TEST::run).doesNotThrowAnyException();
    }

    @Test
    void should_not_throw_exception_method_call() {
        JSONObject config = new JSONObject().put("methods", new JSONArray(Collections.singletonList(TestClass.class.getName() + "#run")))
                .put("erroneousCallRate", 0);
        JavaMethodExceptionInstrumentation attack = new JavaMethodExceptionInstrumentation(INSTRUMENTATION, config);

        assertThatCode(TEST::run).doesNotThrowAnyException();
        attack.install();
        assertThatCode(TEST::run).doesNotThrowAnyException();
        attack.reset();
        assertThatCode(TEST::run).doesNotThrowAnyException();
    }

    public static class TestClass {
        private String run() {
            return "HelloWorld";
        }
    }
}