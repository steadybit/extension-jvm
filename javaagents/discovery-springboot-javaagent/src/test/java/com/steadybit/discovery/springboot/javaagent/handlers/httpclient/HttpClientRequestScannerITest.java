/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.httpclient;

import com.netflix.hystrix.HystrixCommand;
import com.netflix.hystrix.HystrixCommandGroupKey;
import com.netflix.hystrix.HystrixObservableCommand;
import io.github.resilience4j.circuitbreaker.CircuitBreaker;
import io.github.resilience4j.circuitbreaker.CircuitBreakerConfig;
import io.github.resilience4j.reactor.circuitbreaker.operator.CircuitBreakerOperator;
import net.bytebuddy.agent.ByteBuddyAgent;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.http.client.HttpComponentsClientHttpRequestFactory;
import org.springframework.web.client.RestTemplate;
import org.springframework.web.reactive.function.client.WebClient;
import reactor.core.publisher.Mono;
import rx.Observable;

import java.lang.instrument.Instrumentation;

import static org.assertj.core.api.Assertions.assertThat;

class HttpClientRequestScannerITest {
    private static final Instrumentation INSTRUMENTATION = ByteBuddyAgent.install();
    private static final CircuitBreaker r4jcircuitBreaker = CircuitBreaker.of("cb-r4j", CircuitBreakerConfig.custom().build());
    private HttpClientRequestScanner scanner;

    @BeforeEach
    void setUp() {
        this.scanner = new HttpClientRequestScanner(INSTRUMENTATION);
        this.scanner.install();
    }

    @AfterEach
    void tearDown() {
        this.scanner.reset();
    }

    @Test
    void should_capture_requests_from_resttemplate() {
        //given
        RestTemplate restTemplate = new RestTemplate();

        //when
        try {
            r4jcircuitBreaker.executeSupplier(() -> restTemplate.getForObject("http://localhost:8080", Void.class));
        } catch (Exception e) {
            //ignore
        }

        try {
            new HystrixCommand<Void>(HystrixCommandGroupKey.Factory.asKey("cb-hystrix")) {
                @Override
                protected Void run() {
                    return restTemplate.getForObject("http://localhost:9090", Void.class);
                }

                @Override
                protected Void getFallback() {
                    return null;
                }
            }.execute();
        } catch (Exception e) {
            //ignore
        }

        HttpComponentsClientHttpRequestFactory requestFactory = new HttpComponentsClientHttpRequestFactory();
        requestFactory.setReadTimeout(250);
        RestTemplate restTemplateWithTimeout = new RestTemplate(requestFactory);
        try {
            restTemplateWithTimeout.getForObject("http://localhost", Void.class);
        } catch (Exception e) {
            //ignore
        }

        //then
        assertThat(this.scanner.getRequests()).hasSize(3).anySatisfy(d -> {
            assertThat(d.getAddress()).isEqualTo("localhost:80");
            assertThat(d.getScheme()).isEqualTo("http");
            assertThat(d.getTimeout()).isEqualTo(250);
            assertThat(d.hasCircuitBreaker()).isFalse();
        }).anySatisfy(d -> {
            assertThat(d.getAddress()).isEqualTo("localhost:8080");
            assertThat(d.getScheme()).isEqualTo("http");
            assertThat(d.getTimeout()).isZero();
            assertThat(d.hasCircuitBreaker()).isTrue();
        }).anySatisfy(d -> {
            assertThat(d.getAddress()).isEqualTo("localhost:9090");
            assertThat(d.getScheme()).isEqualTo("http");
            assertThat(d.getTimeout()).isZero();
            assertThat(d.hasCircuitBreaker()).isTrue();
        });
    }

    @Test
    void should_capture_addresses_from_webclient() {
        //given
        WebClient webClient = WebClient.builder().build();

        //when
        webClient.get().uri("http://localhost").retrieve().toBodilessEntity().onErrorResume((e) -> Mono.empty()).block();

        webClient.get().uri("http://localhost:8080").retrieve()
                .toBodilessEntity().onErrorResume((e) -> Mono.empty())
                .transformDeferred(CircuitBreakerOperator.of(r4jcircuitBreaker))
                .block();

        new HystrixObservableCommand<Object>(HystrixCommandGroupKey.Factory.asKey("cb-hystrix")) {
            @Override
            protected Observable<Object> construct() {
                return Observable.from(webClient.get().uri("http://localhost:9090").retrieve().toBodilessEntity().toFuture());
            }
        }.observe().onErrorResumeNext(Observable.empty()).toBlocking().singleOrDefault(null);

        //then
        assertThat(this.scanner.getRequests()).anySatisfy(d -> {
            assertThat(d.getAddress()).isEqualTo("localhost:80");
            assertThat(d.getScheme()).isEqualTo("http");
            assertThat(d.getTimeout()).isZero();
            assertThat(d.hasCircuitBreaker()).isFalse();
        }).anySatisfy(d -> {
            assertThat(d.getAddress()).isEqualTo("localhost:8080");
            assertThat(d.getScheme()).isEqualTo("http");
            assertThat(d.getTimeout()).isZero();
            assertThat(d.hasCircuitBreaker()).isTrue();
        }).anySatisfy(d -> {
            assertThat(d.getAddress()).isEqualTo("localhost:9090");
            assertThat(d.getScheme()).isEqualTo("http");
            assertThat(d.getTimeout()).isZero();
            assertThat(d.hasCircuitBreaker()).isTrue();
        });
    }
}