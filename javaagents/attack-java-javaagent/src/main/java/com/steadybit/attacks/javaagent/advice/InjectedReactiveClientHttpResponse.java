/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.advice;

import org.springframework.core.io.buffer.DataBuffer;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatus;
import org.springframework.http.ResponseCookie;
import org.springframework.http.client.reactive.ClientHttpResponse;
import org.springframework.util.LinkedMultiValueMap;
import org.springframework.util.MultiValueMap;
import reactor.core.publisher.Flux;

public class InjectedReactiveClientHttpResponse implements ClientHttpResponse {
    private final int httpStatus;

    public InjectedReactiveClientHttpResponse(int httpStatus) {
        this.httpStatus = httpStatus;
    }

    @Override
    public HttpStatus getStatusCode() {
        return HttpStatus.resolve(this.getRawStatusCode());
    }

    @Override
    public int getRawStatusCode() {
        return this.httpStatus;
    }

    @Override
    public MultiValueMap<String, ResponseCookie> getCookies() {
        return new LinkedMultiValueMap<>();
    }

    @Override
    public Flux<DataBuffer> getBody() {
        return Flux.empty();
    }

    @Override
    public HttpHeaders getHeaders() {
        return new HttpHeaders();
    }
}
