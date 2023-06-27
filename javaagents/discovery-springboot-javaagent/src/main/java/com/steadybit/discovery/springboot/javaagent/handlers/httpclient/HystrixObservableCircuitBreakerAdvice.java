/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.httpclient;

import com.steadybit.javaagent.instrumentation.InstrumentationPluginDispatcher;
import com.steadybit.javaagent.instrumentation.Registration;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import rx.Observable;

public class HystrixObservableCircuitBreakerAdvice {
    @Advice.OnMethodExit(suppress = Throwable.class)
    static void exit(@Registration int registration, @Advice.Return(readOnly = false) Observable<?> result) {
        result = (Observable<?>) InstrumentationPluginDispatcher.find(registration).exec(12, result);
    }
}
