/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.httpclient;

import com.steadybit.javaagent.instrumentation.InstrumentationPluginDispatcher;
import com.steadybit.javaagent.instrumentation.Registration;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import org.springframework.http.client.reactive.ClientHttpResponse;
import reactor.core.publisher.Mono;

import java.net.URI;

public class WebClientAdvice {
    @Advice.OnMethodExit(suppress = Throwable.class)
    static void exit(@Registration int registration, @Advice.Argument(1) URI uri, @Advice.Return(readOnly = false) Mono<ClientHttpResponse> result) {
        result = (Mono<ClientHttpResponse>) InstrumentationPluginDispatcher.find(registration).exec(0, uri, result);
    }
}
