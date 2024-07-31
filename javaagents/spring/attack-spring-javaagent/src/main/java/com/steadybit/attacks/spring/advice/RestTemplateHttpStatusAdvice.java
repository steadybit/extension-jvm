/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.spring.advice;

import com.steadybit.javaagent.instrumentation.InstrumentationPluginDispatcher;
import com.steadybit.javaagent.instrumentation.Registration;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import org.springframework.http.HttpMethod;
import org.springframework.http.client.ClientHttpRequest;
import org.springframework.http.client.ClientHttpResponse;

import java.io.IOException;
import java.net.SocketTimeoutException;
import java.net.URI;

public class RestTemplateHttpStatusAdvice {
    @Advice.OnMethodEnter(skipOn = Integer.class)
    static Integer enter(@Registration int registration,
            @Advice.This ClientHttpRequest request) {
        URI uri = request.getURI();
        HttpMethod method = request.getMethod();
        return (Integer) InstrumentationPluginDispatcher
                .find(registration)
                .exec(1, method != null ? method.toString() : null, uri.getHost(), uri.getPort(), uri.getPath());
    }

    @Advice.OnMethodExit
    // java:S1226 This is how bytebuddy assigns new response values.
    // java:S2095 it's on purpose
    @SuppressWarnings({ "java:S1226", "java:S2095" })
    static void exit(@Advice.Return(readOnly = false) ClientHttpResponse response, @Advice.Enter Integer simulatedStatus) throws IOException {
        if (simulatedStatus == null) {
            return;
        } else if (simulatedStatus > 0) {
            response = new InjectedClientHttpResponse(simulatedStatus);
        } else if (simulatedStatus == -1) {
            throw new SocketTimeoutException("Simulated socket timeout through a scheduled Steadybit experiment.");
        } else if (simulatedStatus == -2) {
            throw new IOException("Simulated connection/HTTP protocol error through a scheduled Steadybit experiment.");
        }
    }
}
