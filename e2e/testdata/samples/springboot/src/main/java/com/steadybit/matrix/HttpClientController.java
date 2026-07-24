/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package com.steadybit.matrix;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestParam;
import org.springframework.web.bind.annotation.RestController;
import org.springframework.web.client.RestTemplate;
import org.springframework.web.reactive.function.client.WebClient;
import org.springframework.web.reactive.function.client.WebClientResponseException;

@RestController
public class HttpClientController {

    private final String downstreamUrl;
    private final RestTemplate restTemplate = new RestTemplate();
    private final WebClient webClient = WebClient.create();

    public HttpClientController(@Value("${downstream.url}") String downstreamUrl) {
        this.downstreamUrl = downstreamUrl;
    }

    @GetMapping("/http/resttemplate")
    public String restTemplate(@RequestParam(name = "url", required = false) String url) {
        return "resttemplate:" + restTemplate.getForObject(target(url), String.class);
    }

    // Surfaces the downstream status so a test can assert the *exact* injected status code
    // (WebClient would otherwise let the exception surface as a generic 500).
    @GetMapping("/http/webclient")
    public ResponseEntity<String> webClient(@RequestParam(name = "url", required = false) String url) {
        try {
            String body = webClient.get().uri(target(url)).retrieve().bodyToMono(String.class).block();
            return ResponseEntity.ok("webclient:" + body);
        } catch (WebClientResponseException e) {
            return ResponseEntity.status(e.getStatusCode().value()).body("webclient-error:" + e.getStatusCode().value());
        }
    }

    private String target(String url) {
        return (url == null || url.isEmpty()) ? downstreamUrl : url;
    }
}
