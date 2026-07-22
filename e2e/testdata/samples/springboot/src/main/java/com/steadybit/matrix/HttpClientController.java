/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package com.steadybit.matrix;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.client.RestTemplate;
import org.springframework.web.reactive.function.client.WebClient;

@RestController
public class HttpClientController {

    private final String downstreamUrl;
    private final RestTemplate restTemplate = new RestTemplate();
    private final WebClient webClient = WebClient.create();

    public HttpClientController(@Value("${downstream.url}") String downstreamUrl) {
        this.downstreamUrl = downstreamUrl;
    }

    @GetMapping("/http/resttemplate")
    public String restTemplate() {
        return "resttemplate:" + restTemplate.getForObject(downstreamUrl, String.class);
    }

    @GetMapping("/http/webclient")
    public String webClient() {
        return "webclient:" + webClient.get().uri(downstreamUrl).retrieve().bodyToMono(String.class).block();
    }
}
