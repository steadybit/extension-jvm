/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.advice;

import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import okhttp3.OkHttpClient;
import org.apache.http.client.config.RequestConfig;
import org.apache.http.client.protocol.HttpClientContext;
import org.apache.http.protocol.HttpContext;
import org.springframework.http.client.ClientHttpRequest;

import java.lang.reflect.Field;
import java.net.HttpURLConnection;
import java.net.SocketTimeoutException;
import java.net.URI;
import java.util.concurrent.ThreadLocalRandom;

public class RestTemplateDelayAdvice {
    @Advice.OnMethodExit
    static void exit(@Delay long delay, @Jitter boolean delayJitter, @HostAddress String hostAddress, @Advice.This ClientHttpRequest request)
            throws SocketTimeoutException {

        if (!hostAddress.equals("*")) {
            URI uri = request.getURI();
            String requestHostAddress = uri.getPort() == -1 ? uri.getHost() : uri.getHost() + ":" + uri.getPort();

            if (!requestHostAddress.equalsIgnoreCase(hostAddress)) {
                return;
            }
        }

        long millis;
        if (delayJitter) {
            double jitterValue = 1.3d - ThreadLocalRandom.current().nextDouble(0.6d);
            millis = Math.round(jitterValue * delay);
        } else {
            millis = delay;
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