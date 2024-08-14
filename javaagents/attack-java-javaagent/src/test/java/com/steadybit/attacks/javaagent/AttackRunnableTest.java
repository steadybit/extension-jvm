/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent;

import com.github.tomakehurst.wiremock.WireMockServer;
import com.steadybit.javaagent.log.RemoteAgentLogger;
import org.json.JSONObject;
import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.ArrayList;
import java.util.List;

import static com.github.tomakehurst.wiremock.client.WireMock.aResponse;
import static com.github.tomakehurst.wiremock.client.WireMock.get;
import static com.github.tomakehurst.wiremock.client.WireMock.post;
import static com.github.tomakehurst.wiremock.client.WireMock.postRequestedFor;
import static com.github.tomakehurst.wiremock.client.WireMock.urlEqualTo;
import static com.github.tomakehurst.wiremock.core.WireMockConfiguration.options;
import static com.github.tomakehurst.wiremock.stubbing.Scenario.STARTED;
import static java.util.Collections.synchronizedList;
import static org.assertj.core.api.Assertions.assertThat;

class AttackRunnableTest {
    private static WireMockServer wireMock;
    private static final List<String> events = synchronizedList(new ArrayList<>());
    private AttackRunnable attackRunnable;

    @BeforeAll
    static void beforeAll() {
        RemoteAgentLogger.setLogToSystem(true);
        wireMock = new WireMockServer(options().dynamicPort());
        wireMock.start();
    }

    @AfterAll
    static void afterAll() {
        wireMock.stop();
    }

    @BeforeEach
    void setUp() {
        this.attackRunnable = new AttackRunnable(null, "http://localhost:" + wireMock.port());
        events.clear();
        wireMock.resetAll();
    }

    @Test
    void should_run_java_attack() {
        //given
        wireMock.stubFor(get("/").willReturn(
                aResponse().withStatus(200).withBody("{ \"attack-class\":\"" + TestInstrumentationAttack.class.getName() + "\", \"duration\": 1000}")));
        wireMock.stubFor(post("/started").willReturn(aResponse().withStatus(200)));
        wireMock.stubFor(post("/stopped").willReturn(aResponse().withStatus(200)));

        //when
        long start = System.currentTimeMillis();
        this.attackRunnable.run();
        long duration = System.currentTimeMillis() - start;

        //then
        assertThat(duration).isGreaterThanOrEqualTo(1000);
        assertThat(events).contains("install", "reset");

        wireMock.verify(0, postRequestedFor(urlEqualTo("/failed")));
    }

    @Test
    void should_stop_java_attack() {
        //given
        wireMock.stubFor(get("/").inScenario("fault")
                .whenScenarioStateIs(STARTED)
                .willSetStateTo("RETURN-FAULT")
                .willReturn(
                        aResponse().withStatus(200).withBody("{ \"attack-class\":\"" + TestInstrumentationAttack.class.getName() + "\", \"duration\": 10000}")));

        wireMock.stubFor(get("/").inScenario("fault").whenScenarioStateIs("RETURN_FAULT").willReturn(aResponse().withStatus(500)));
        wireMock.stubFor(post("/started").willReturn(aResponse().withStatus(200)));
        wireMock.stubFor(post("/stopped").willReturn(aResponse().withStatus(200)));

        //when
        long start = System.currentTimeMillis();
        this.attackRunnable.run();
        long duration = System.currentTimeMillis() - start;

        //then
        assertThat(duration).isLessThan(10000L);
        assertThat(events).contains("install", "reset");

        wireMock.verify(0, postRequestedFor(urlEqualTo("/failed")));
    }

    @Test
    void should_report_failed_java_attack() {
        //given
        wireMock.stubFor(get("/").willReturn(
                aResponse().withStatus(200).withBody("{ \"attack-class\":\"" + FailureTestInstrumentationAttack.class.getName() + "\", \"duration\": 100}")));

        //when
        this.attackRunnable.run();

        //then
        wireMock.verify(1, postRequestedFor(urlEqualTo("/failed")));
    }

    public static class TestInstrumentationAttack implements Installable {
        public TestInstrumentationAttack(java.lang.instrument.Instrumentation instrumentation, JSONObject config) {
        }

        @Override
        public void install() {
            events.add("install");
        }

        @Override
        public void reset() {
            events.add("reset");
        }
    }

    public static class FailureTestInstrumentationAttack implements Installable {
        public FailureTestInstrumentationAttack(java.lang.instrument.Instrumentation instrumentation, JSONObject config) {
        }

        @Override
        public void install() {
            throw new RuntimeException("Test");
        }

        @Override
        public void reset() {
        }
    }
}