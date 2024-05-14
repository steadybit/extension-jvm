/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.httpclient;

import com.steadybit.javaagent.instrumentation.InstrumentationPluginDispatcher;
import com.steadybit.javaagent.instrumentation.Registration;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;

public class HystrixCircuitBreakerAdvice {
    @Advice.OnMethodEnter(suppress = Throwable.class)
    static void enter(@Registration int registration) {
        InstrumentationPluginDispatcher.find(registration).exec(10);
    }

    @Advice.OnMethodExit(suppress = Throwable.class)
    static void exit(@Registration int registration) {
        InstrumentationPluginDispatcher.find(registration).exec(11);
    }
}
