/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.advice;

import org.springframework.core.io.buffer.DataBuffer;
import org.springframework.http.HttpHeaders;
import org.springframework.http.HttpStatusCode;
import org.springframework.http.ResponseCookie;
import org.springframework.http.client.reactive.ClientHttpResponse;
import org.springframework.util.LinkedMultiValueMap;
import org.springframework.util.MultiValueMap;
import reactor.core.publisher.Flux;

/**
 * Spring Framework 6 variant of the injected reactive client-http response, used by
 * {@link WebClientHttpStatusAdviceSpring6}. Spring 6 changed {@code ClientHttpResponse#getStatusCode()} to
 * return {@code HttpStatusCode}, which the Spring-5 class cannot implement, so this variant is injected into
 * Spring 6+ targets.
 */
public class InjectedReactiveClientHttpResponseSpring6 implements ClientHttpResponse {
    private final int httpStatus;

    public InjectedReactiveClientHttpResponseSpring6(int httpStatus) {
        this.httpStatus = httpStatus;
    }

    @Override
    public HttpStatusCode getStatusCode() {
        return HttpStatusCode.valueOf(this.httpStatus);
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
