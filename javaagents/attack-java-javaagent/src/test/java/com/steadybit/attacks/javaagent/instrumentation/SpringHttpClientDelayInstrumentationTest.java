/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import org.json.JSONArray;
import org.json.JSONObject;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.lang.instrument.Instrumentation;
import java.util.Arrays;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;

class SpringHttpClientDelayInstrumentationTest {

    private JSONObject config;

    @BeforeEach
    void beforeEach() {
        this.config = new JSONObject();
    }

    Long attack(String httpMethod, String host, int port, String path) {
        return (Long) new SpringHttpClientDelayInstrumentation(mock(Instrumentation.class), this.config).exec(2, httpMethod, host, port, path);
    }

    @Test
    void should_execute_a_500ms_delay_attack_by_default() {
        assertThat(this.attack("POST", "example.com", 443, "/")).isEqualTo(500);
    }

    @Test
    void should_support_restriction_to_specific_methods() {
        this.config.put("httpMethods", this.toArray("POST", "PUT"));
        assertThat(this.attack("GET", "example.com", 443, "/")).isNull();
        assertThat(this.attack("post", "example.com", 443, "/")).isEqualTo(500);
        assertThat(this.attack("put", "example.com", 443, "/")).isEqualTo(500);
    }

    @Test
    void should_support_empty_methods_arrays() {
        this.config.put("httpMethods", this.toArray());
        assertThat(this.attack("GET", "example.com", 443, "/")).isEqualTo(500);
        assertThat(this.attack("post", "example.com", 443, "/")).isEqualTo(500);
        assertThat(this.attack("put", "example.com", 443, "/")).isEqualTo(500);
    }

    @Test
    void should_support_host_matching_requiring_port() {
        this.config.put("hostAddress", "eXample.com");
        assertThat(this.attack("post", "example.com", 443, "/")).isNull();
        assertThat(this.attack("post", "example.com", -1, "/")).isEqualTo(500);

        this.config.put("hostAddress", "eXample.com:443");
        assertThat(this.attack("post", "example.com", 443, "/")).isEqualTo(500);
        assertThat(this.attack("post", "example.com", -1, "/")).isNull();
    }

    @Test
    void should_support_asterisk_host() {
        this.config.put("hostAddress", "*");
        assertThat(this.attack("GET", "example.com", 443, "/")).isEqualTo(500);
        assertThat(this.attack("post", "acme.example.com", -1, "/")).isEqualTo(500);
    }

    @Test
    void should_support_path_matching() {
        this.config.put("urlPath", "/api");
        assertThat(this.attack("post", "example.com", 443, "/aPi")).isEqualTo(500);
        assertThat(this.attack("post", "example.com", -1, "/")).isEqualTo(500);
        assertThat(this.attack("post", "example.com", -1, "/api/attacks")).isNull();
    }

    @Test
    void should_support_asterisk_path() {
        this.config.put("urlPath", "*");
        assertThat(this.attack("post", "example.com", 443, "/aPi")).isEqualTo(500);
        assertThat(this.attack("post", "example.com", -1, "/")).isEqualTo(500);
        assertThat(this.attack("post", "example.com", -1, "/api/attacks")).isEqualTo(500);
    }

    @Test
    void should_support_empty_as_asterisk_path() {
        this.config.remove("urlPath");
        assertThat(this.attack("post", "example.com", 443, "/aPi")).isEqualTo(500);
        assertThat(this.attack("post", "example.com", -1, "/")).isEqualTo(500);
        assertThat(this.attack("post", "example.com", -1, "/api/attacks")).isEqualTo(500);
    }

    @Test
    void should_support_empty_as_asterisk_path_2() {
        this.config.put("urlPath", "");
        assertThat(this.attack("post", "example.com", 443, "/aPi")).isEqualTo(500);
        assertThat(this.attack("post", "example.com", -1, "/")).isEqualTo(500);
        assertThat(this.attack("post", "example.com", -1, "/api/attacks")).isEqualTo(500);
    }


    private JSONArray toArray(String... values) {
        return new JSONArray(Arrays.asList(values));
    }


}