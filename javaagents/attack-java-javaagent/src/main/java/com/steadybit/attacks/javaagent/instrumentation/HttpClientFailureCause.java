/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

/**
 * Modelled after https://nginx.org/en/docs/http/ngx_http_proxy_module.html#proxy_next_upstream
 */
public class HttpClientFailureCause {
    public static final String ERROR = "ERROR";
    public static final String TIMEOUT = "TIMEOUT";
    public static final String HTTP_500 = "HTTP_500";
    public static final String HTTP_502 = "HTTP_502";
    public static final String HTTP_503 = "HTTP_503";
    public static final String HTTP_504 = "HTTP_504";
    public static final String HTTP_5XX = "HTTP_5XX";
    public static final String HTTP_400 = "HTTP_400";
    public static final String HTTP_403 = "HTTP_403";
    public static final String HTTP_404 = "HTTP_404";
    public static final String HTTP_429 = "HTTP_429";
    public static final String HTTP_4XX = "HTTP_4XX";
}
