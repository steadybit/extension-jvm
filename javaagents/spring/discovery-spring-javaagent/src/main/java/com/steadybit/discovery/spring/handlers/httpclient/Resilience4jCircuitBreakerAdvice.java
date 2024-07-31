/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.spring.handlers.httpclient;

import com.steadybit.javaagent.instrumentation.InstrumentationPluginDispatcher;
import com.steadybit.javaagent.instrumentation.Registration;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import com.steadybit.shaded.net.bytebuddy.implementation.bytecode.assign.Assigner;

public class Resilience4jCircuitBreakerAdvice {
    @Advice.OnMethodExit(suppress = Throwable.class) static void exit(@Registration int registration, @Advice.Origin("#m") String method,
            @Advice.Return(typing = Assigner.Typing.DYNAMIC) Object result) {
        switch (method) {
        case "tryAcquirePermission":
            if (Boolean.TRUE.equals(result)) {
                InstrumentationPluginDispatcher.find(registration).exec(10);
            }
            break;

        case "acquirePermission":
            InstrumentationPluginDispatcher.find(registration).exec(10);
            break;

        case "onSuccess":
        case "onError":
        case "releasePermission":
            InstrumentationPluginDispatcher.find(registration).exec(11);
            break;
        default:
            break;
        }
    }
}
