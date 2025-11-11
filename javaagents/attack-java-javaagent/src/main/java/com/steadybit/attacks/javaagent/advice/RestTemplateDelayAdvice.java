/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.advice;

import com.steadybit.javaagent.instrumentation.InstrumentationPluginDispatcher;
import com.steadybit.javaagent.instrumentation.Registration;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import okhttp3.OkHttpClient;
import org.apache.http.client.config.RequestConfig;
import org.apache.http.client.protocol.HttpClientContext;
import org.apache.http.protocol.HttpContext;
import org.springframework.http.HttpMethod;
import org.springframework.http.client.ClientHttpRequest;

import java.lang.reflect.Field;
import java.net.HttpURLConnection;
import java.net.SocketTimeoutException;
import java.net.URI;

public class RestTemplateDelayAdvice {
    @Advice.OnMethodExit
    static void exit(@Registration int registration, @Advice.This ClientHttpRequest request) throws SocketTimeoutException {
        URI uri = request.getURI();
        HttpMethod method = request.getMethod();

        Long millis = (Long) InstrumentationPluginDispatcher
                .find(registration)
                .exec(2, method != null ? method.toString() : null, uri.getHost(), uri.getPort(), uri.getPath());
        if (millis == null) {
            return;
        }

        int readTimeout = 0;
        Class<? extends ClientHttpRequest> requestClass = request.getClass();

        if (requestClass.getName().startsWith("org.springframework.http.client.Simple")) {
            // read socket timeout for simple jdk http client
            try {
                Field connectionField = requestClass.getDeclaredField("connection");
                connectionField.setAccessible(true);
                Object connection = connectionField.get(request);
                if (connection instanceof HttpURLConnection) {
                    readTimeout = ((HttpURLConnection) connection).getReadTimeout();
                }
            } catch (Exception e) {
                //ignore
            }
        } else if (requestClass.getName().startsWith("org.springframework.http.client.HttpComponents")) {
            try {
                Field httpContextField = requestClass.getDeclaredField("httpContext");
                httpContextField.setAccessible(true);
                Object httpContext = httpContextField.get(request);
                if (httpContext instanceof HttpContext) {
                    Object requestConfig = ((HttpContext) httpContext).getAttribute(HttpClientContext.REQUEST_CONFIG);
                    if (requestConfig instanceof RequestConfig) {
                        readTimeout = ((RequestConfig) requestConfig).getSocketTimeout();
                    } else {
                        readTimeout = RequestConfig.DEFAULT.getSocketTimeout();
                    }
                }
            } catch (Exception e) {
                //ignore
            }
        } else if (requestClass.getName().startsWith("org.springframework.http.client.OkHttp3")) {
            try {
                // read socket timeout for OkHttp3
                Field clientField = requestClass.getDeclaredField("client");
                clientField.setAccessible(true);
                Object client = clientField.get(request);
                if (client instanceof OkHttpClient) {
                    readTimeout = ((OkHttpClient) client).readTimeoutMillis();
                }
            } catch (Exception e) {
                //ignore
            }
        }

        try {
            //if the delay is longer than the configured timeout (0=infinite) we need throw an exception to mimic the real world.
            if (readTimeout == 0 || readTimeout >= millis) {
                Thread.sleep(millis);
            } else {
                Thread.sleep(readTimeout);
                throw new SocketTimeoutException();
            }
        } catch (InterruptedException e) {
            //ignore the interruption and restore interruption flag.
            Thread.currentThread().interrupt();
        }
    }
}