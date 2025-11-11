/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.advice;

import com.steadybit.javaagent.instrumentation.InstrumentationPluginDispatcher;
import com.steadybit.javaagent.instrumentation.Registration;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import org.springframework.http.HttpMethod;
import org.springframework.http.client.reactive.ClientHttpResponse;
import reactor.core.publisher.Mono;

import java.net.URI;
import java.time.Duration;

public class WebClientDelayAdvice {
    @Advice.OnMethodExit
    static void exit(@Registration int registration, @Advice.Argument(0) HttpMethod httpMethod, @Advice.Argument(1) URI uri,
                     @Advice.Return(readOnly = false) Mono<ClientHttpResponse> response) {

        Long millis = (Long) InstrumentationPluginDispatcher
                .find(registration)
                .exec(2, httpMethod != null ? httpMethod.toString() : null, uri.getHost(), uri.getPort(), uri.getPath());
        if (millis == null) {
            return;
        }

        response = Mono.delay(Duration.ofMillis(millis)).then(response);
    }
}