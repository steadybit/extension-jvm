/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent;

import java.lang.instrument.Instrumentation;

public class TestAgentV2 {

    public static void init(String agentArguments, Instrumentation instrumentation, ClassLoader previousAgent) throws Exception {
        if (previousAgent != null) {
            previousAgent.loadClass("com.steadybit.javaagent.TestAgentV1").getMethod("stop").invoke(null);
        }
        TrampolineAgentTest.events.add("Loaded TestAgentV2");
    }
}
