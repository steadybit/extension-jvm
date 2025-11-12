/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.advice;

import com.steadybit.javaagent.instrumentation.InstrumentationPluginDispatcher;
import com.steadybit.javaagent.instrumentation.Registration;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import org.springframework.http.HttpMethod;
import org.springframework.http.client.reactive.ClientHttpResponse;
import reactor.core.publisher.Mono;

import java.io.IOException;
import java.net.SocketTimeoutException;
import java.net.URI;

public class WebClientHttpStatusAdvice {
    @Advice.OnMethodEnter(skipOn = Integer.class)
    static Integer enter(@Registration int registration, @Advice.Argument(0) HttpMethod httpMethod, @Advice.Argument(1) URI uri) {
        return (Integer) InstrumentationPluginDispatcher
                .find(registration)
                .exec(1, httpMethod != null ? httpMethod.toString() : null, uri);
    }

    @Advice.OnMethodExit
    @SuppressWarnings("java:S1226") // This is how bytebuddy assigns new response values.
    static void exit(@Advice.Return(readOnly = false) Mono<ClientHttpResponse> response, @Advice.Enter Integer simulatedStatus) {
        if (simulatedStatus == null) {
            return;
        } else if (simulatedStatus > 0) {
            response = Mono.just(new InjectedReactiveClientHttpResponse(simulatedStatus));
        } else if (simulatedStatus == -1) {
            response = Mono.error(new SocketTimeoutException("Simulated socket timeout through a scheduled Steadybit experiment."));
        } else if (simulatedStatus == -2) {
            response = Mono.error(new IOException("Simulated connection/HTTP protocol error through a scheduled Steadybit experiment."));
        }
    }
}