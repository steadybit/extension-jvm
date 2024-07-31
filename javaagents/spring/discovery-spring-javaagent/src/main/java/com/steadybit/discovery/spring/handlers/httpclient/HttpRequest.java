/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.spring.handlers.httpclient;

public class HttpRequest {
    private final String address;
    private final String scheme;
    private final boolean hasCircuitBreaker;
    private final int timeout;

    public HttpRequest(String address, String scheme, boolean hasCircuitBreaker, int timeout) {
        this.address = address;
        this.scheme = scheme;
        this.hasCircuitBreaker = hasCircuitBreaker;
        this.timeout = timeout;
    }

    public String getAddress() {
        return this.address;
    }

    public String getScheme() {
        return this.scheme;
    }

    public boolean hasCircuitBreaker() {
        return this.hasCircuitBreaker;
    }

    public int getTimeout() {
        return this.timeout;
    }

    @Override
    public String toString() {
        return "HttpRequest{" +
                "address='" + this.address + '\'' +
                ", scheme='" + this.scheme + '\'' +
                ", hasCircuitBreaker=" + this.hasCircuitBreaker +
                ", timeout=" + this.timeout +
                '}';
    }
}
