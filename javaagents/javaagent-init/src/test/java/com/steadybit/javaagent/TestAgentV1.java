/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent;

import java.lang.instrument.Instrumentation;

public class TestAgentV1 {

    public static void init(String agentArguments, Instrumentation instrumentation, ClassLoader previousAgent) {
        TrampolineAgentTest.events.add("Loaded TestAgentV1");
    }

    public static void stop() {
        TrampolineAgentTest.events.add("Stopped TestAgentV1");
    }
}
