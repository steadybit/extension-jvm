/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.spring.handlers.httpclient;

import com.steadybit.javaagent.instrumentation.InstrumentationPluginDispatcher;
import com.steadybit.javaagent.instrumentation.Registration;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import org.springframework.http.client.ClientHttpRequest;

public class RestTemplateAdvice {
    @Advice.OnMethodEnter(suppress = Throwable.class)
    static void enter(@Registration int registration, @Advice.This ClientHttpRequest request) {
        InstrumentationPluginDispatcher.find(registration).exec(1, request);
    }
}
