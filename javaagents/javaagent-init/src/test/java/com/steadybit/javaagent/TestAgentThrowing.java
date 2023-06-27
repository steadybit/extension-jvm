/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent;

import java.lang.instrument.Instrumentation;

public class TestAgentThrowing {

    public static void init(String agentArguments, Instrumentation instrumentation, ClassLoader previousAgent) {
        throw new RuntimeException("test");
    }

    public static void stop() {
        TrampolineAgentTest.events.add("Stopped TestAgentV1");
    }
}
