/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.springboot2.advice;

import org.springframework.http.HttpHeaders;
import org.springframework.http.client.AbstractClientHttpResponse;

import java.io.ByteArrayInputStream;
import java.io.InputStream;

public class InjectedClientHttpResponse extends AbstractClientHttpResponse {
    private final int httpStatus;

    public InjectedClientHttpResponse(int httpStatus) {
        this.httpStatus = httpStatus;
    }

    @Override
    public int getRawStatusCode() {
        return this.httpStatus;
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
