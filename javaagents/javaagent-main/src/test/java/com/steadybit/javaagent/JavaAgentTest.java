/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent;

import com.github.tomakehurst.wiremock.WireMockServer;

import static com.github.tomakehurst.wiremock.client.WireMock.ok;
import static com.github.tomakehurst.wiremock.client.WireMock.put;
import static com.github.tomakehurst.wiremock.client.WireMock.putRequestedFor;
import static com.github.tomakehurst.wiremock.client.WireMock.urlMatching;
import static com.github.tomakehurst.wiremock.core.WireMockConfiguration.options;

import net.bytebuddy.agent.ByteBuddyAgent;

import static org.awaitility.Awaitility.await;

import org.junit.jupiter.api.AfterAll;
import org.junit.jupiter.api.BeforeAll;
import org.junit.jupiter.api.Test;

import java.time.Duration;

class JavaAgentTest {
    private static WireMockServer wireMock;

    @BeforeAll
    static void setup() {
        wireMock = new WireMockServer(options().dynamicPort());
        wireMock.start();
    }

    @AfterAll
    static void tearDown() {
        JavaAgent.stop();
        wireMock.stop();
    }

    @Test
    void should_call_agent_main_without_heartbeat() throws Exception {
        //Given
        wireMock.stubFor(put("/javaagent").willReturn(ok()));

        System.setProperty("STEADYBIT_LOG_JAVAAGENT_STDOUT", "true");
        System.setProperty("STEADYBIT_LOG_LEVEL", "DEBUG");

        //When
        JavaAgent.init("disableBootstrapLoaderInjection=true,pid=6,host=127.0.0.1,port=" + wireMock.port(), ByteBuddyAgent.install(), null);
        //Then
        await().untilAsserted(() -> wireMock.verify(putRequestedFor(urlMatching("/javaagent")))
        );
    }

    @Test
    void should_not_call_agent_main_with_missing_heartbeat() throws Exception {
        //Given
        wireMock.stubFor(put("/javaagent").willReturn(ok()));

        System.setProperty("STEADYBIT_LOG_JAVAAGENT_STDOUT", "true");
        System.setProperty("STEADYBIT_LOG_LEVEL", "DEBUG");

        //When
        JavaAgent.init("heartbeat=/missing-file,disableBootstrapLoaderInjection=true,pid=6,host=127.0.0.1,port=" + wireMock.port(), ByteBuddyAgent.install(), null);
        //Then
        await().pollDelay(Duration.ofSeconds(5))
                .untilAsserted(() -> wireMock.verify(0, putRequestedFor(urlMatching("/javaagent"))));
    }

}
