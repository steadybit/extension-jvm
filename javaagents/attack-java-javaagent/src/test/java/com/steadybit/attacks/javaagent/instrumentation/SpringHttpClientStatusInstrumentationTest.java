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
import java.util.HashMap;
import java.util.Map;
import java.util.Set;

import static org.assertj.core.api.Assertions.assertThat;
import static org.mockito.Mockito.mock;

class SpringHttpClientStatusInstrumentationTest {

    private JSONObject config;

    @BeforeEach
    void beforeEach() {
        this.config = new JSONObject();
    }

    Integer attack(String httpMethod, String host, int port, String path) {
        return (Integer) new SpringHttpClientStatusInstrumentation(mock(Instrumentation.class), this.config)
                .exec(1, httpMethod, host, port, path);
    }

    @Test
    void should_execute_a_500_status_code_attack_by_default() {
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
        assertThat(this.attack("post", "example.com", -1, "/")).isNull();
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

    @Test
    void should_support_multiple_failure_scenarios() {
        // Given
        this.config.put("failureCauses", this.toArray(HttpClientFailureCause.ERROR, HttpClientFailureCause.TIMEOUT, HttpClientFailureCause.HTTP_404));

        // When
        Set<Integer> seenErrors = this.collectSeenErrors(200).keySet();

        // Then
        assertThat(seenErrors)
                .hasSize(3)
                .contains(-1, -2, 404);
    }

    @Test
    void should_generate_random_4xx_status_codes() {
        // Given
        this.config.put("failureCauses", this.toArray(HttpClientFailureCause.HTTP_4XX));

        // When
        Set<Integer> seenErrors = this.collectSeenErrors(200).keySet();

        // Then
        assertThat(seenErrors).isNotEmpty().allSatisfy(error -> assertThat(error).isGreaterThanOrEqualTo(400).isLessThan(500));
    }

    @Test
    void should_generate_random_5xx_status_codes() {
        // Given
        this.config.put("failureCauses", this.toArray(HttpClientFailureCause.HTTP_5XX));

        // When
        Set<Integer> seenErrors = this.collectSeenErrors(200).keySet();

        // Then
        assertThat(seenErrors).isNotEmpty().allSatisfy(error -> assertThat(error).isGreaterThanOrEqualTo(500).isLessThan(600));
    }

    @Test
    void should_support_error_rates() {
        // Given
        double errorRate = 0.5;
        int simulations = 200;
        int minNumberOfExpectedOccurrences = (int) (simulations * errorRate * 0.8);
        int maxNumberOfExpectedOccurrences = (int) (simulations * errorRate * 1.2);
        this.config.put("erroneousCallRate", (int) (errorRate * 100));

        // When
        Map<Integer, Integer> seenErrors = this.collectSeenErrors(simulations);

        // Then
        assertThat(seenErrors.get(null)).isBetween(minNumberOfExpectedOccurrences, maxNumberOfExpectedOccurrences);
        assertThat(seenErrors.get(500)).isBetween(minNumberOfExpectedOccurrences, maxNumberOfExpectedOccurrences);
    }

    private JSONArray toArray(String... values) {
        return new JSONArray(Arrays.asList(values));
    }

    /**
     * Map from status code to number of occurrences seen within the simulations.
     */
    private Map<Integer, Integer> collectSeenErrors(int simulations) {
        Map<Integer, Integer> seenErrors = new HashMap<>(simulations);

        for (int i = 0; i < simulations; i++) {
            Integer error = this.attack("post", "example.com", 443, "/api");
            Integer count = seenErrors.get(error);
            if (count == null) {
                count = 0;
            }
            count++;
            seenErrors.put(error, count);
        }

        return seenErrors;
    }

}