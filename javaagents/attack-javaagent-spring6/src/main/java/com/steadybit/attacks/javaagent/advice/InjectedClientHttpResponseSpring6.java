/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.advice;

import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatusCode;
import org.springframework.http.client.ClientHttpResponse;

import java.io.ByteArrayInputStream;
import java.io.InputStream;

/**
 * Spring Framework 6 variant of the injected blocking client-http response, used by
 * {@link RestTemplateHttpStatusAdviceSpring6}. Spring 6 removed {@code AbstractClientHttpResponse} (the
 * Spring-5 base class) and changed {@code getStatusCode()} to return {@code HttpStatusCode}, so this variant
 * implements {@link ClientHttpResponse} directly and is injected into Spring 6+ targets.
 */
public class InjectedClientHttpResponseSpring6 implements ClientHttpResponse {
    private final int httpStatus;

    public InjectedClientHttpResponseSpring6(int httpStatus) {
        this.httpStatus = httpStatus;
    }

    @Override
    public HttpStatusCode getStatusCode() {
        return HttpStatusCode.valueOf(this.httpStatus);
    }

    @Override
    public String getStatusText() {
        return "Injected by steadybit";
    }

    @Override
    public void close() {
        //NOOP
    }

    @Override
    public InputStream getBody() {
        return new ByteArrayInputStream(new byte[0]);
    }

    @Override
    public HttpHeaders getHeaders() {
        return new HttpHeaders();
    }
}
