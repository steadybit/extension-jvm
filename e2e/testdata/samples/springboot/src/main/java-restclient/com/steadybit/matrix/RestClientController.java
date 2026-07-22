/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package com.steadybit.matrix;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.client.RestClient;

@RestController
public class RestClientController {

    private final String downstreamUrl;
    private final RestClient restClient = RestClient.create();

    public RestClientController(@Value("${downstream.url}") String downstreamUrl) {
        this.downstreamUrl = downstreamUrl;
    }

    @GetMapping("/http/restclient")
    public String restClient() {
        return "restclient:" + restClient.get().uri(downstreamUrl).retrieve().body(String.class);
    }
}
