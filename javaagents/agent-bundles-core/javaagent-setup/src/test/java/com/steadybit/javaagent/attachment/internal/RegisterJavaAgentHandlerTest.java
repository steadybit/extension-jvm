/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import com.steadybit.agent.resources.EmbeddedResourceHelper;
import com.steadybit.cri.CriClient;
import com.steadybit.docker.DockerClient;
import com.steadybit.javaagent.attachment.JavaAgentFacade;
import com.steadybit.javaagent.attachment.JavaAgentSetupConfiguration;
import static org.assertj.core.api.Assertions.assertThat;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.reactive.WebFluxTest;
import org.springframework.boot.test.context.TestConfiguration;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.context.annotation.Bean;
import org.springframework.http.MediaType;
import org.springframework.http.server.reactive.ServerHttpRequest;
import org.springframework.http.server.reactive.ServerHttpRequestDecorator;
import org.springframework.test.context.ContextConfiguration;
import org.springframework.test.web.reactive.server.WebTestClient;
import org.springframework.web.server.ServerWebExchangeDecorator;
import org.springframework.web.server.WebFilter;

import java.net.InetSocketAddress;

@WebFluxTest(controllers = RegisterJavaAgentHandler.class)
@ContextConfiguration(classes = { JavaAgentSetupConfiguration.class, RegisterJavaAgentHandlerTest.WebFilterConfig.class })
class RegisterJavaAgentHandlerTest {
    private static final String REMOTE_ADDRESS = "192.168.1.2";

    @Autowired
    RemoteJvmConnections remoteJvmConnections;

    @Autowired
    WebTestClient webTestClient;

    @MockBean
    DockerClient dockerClient;

    @MockBean
    CriClient criClient;

    @MockBean
    JavaAgentFacade javaAgentFacade;

    @MockBean
    EmbeddedResourceHelper embeddedResourceHelper;

    @BeforeEach
    void setUp() {
        this.remoteJvmConnections.clear();
    }

    @Test
    void should_register_remote_connection_with_port_only() {
        this.webTestClient.put().uri("/javaagent")
                .contentType(MediaType.TEXT_PLAIN)
                .bodyValue("1234=8080")
                .exchange().expectStatus().is2xxSuccessful();

        var connection = this.remoteJvmConnections.getConnection(1234);
        assertThat(connection).isNotNull();
        assertThat(connection.getHostString()).isEqualTo(REMOTE_ADDRESS);
        assertThat(connection.getPort()).isEqualTo(8080);
    }

    @Test
    void should_register_remote_connection_with_host_and_port() {
        this.webTestClient
                .put()
                .uri("/javaagent")
                .contentType(MediaType.TEXT_PLAIN)
                .bodyValue("1234=172.0.1.12:45455")
                .exchange().expectStatus().is2xxSuccessful();

        var connection = this.remoteJvmConnections.getConnection(1234);
        assertThat(connection).isNotNull();
        assertThat(connection.getHostString()).isEqualTo("172.0.1.12");
        assertThat(connection.getPort()).isEqualTo(45455);
    }

    @Test
    void should_return_400() {
        this.webTestClient.put().uri("/javaagent")
                .contentType(MediaType.TEXT_PLAIN)
                .bodyValue("FOOBAR")
                .exchange().expectStatus().isBadRequest();

        assertThat(this.remoteJvmConnections.size()).isZero();
    }

    @TestConfiguration
    static class WebFilterConfig {
        @Bean
        WebFilter webFilter() {
            return (exchange, chain) -> {
                final ServerHttpRequest decorated = new ServerHttpRequestDecorator(exchange.getRequest()) {
                    @Override
                    public InetSocketAddress getRemoteAddress() {
                        return new InetSocketAddress(REMOTE_ADDRESS, 80);
                    }
                };

                return chain.filter(new ServerWebExchangeDecorator(exchange) {
                    @Override
                    public ServerHttpRequest getRequest() {
                        return decorated;
                    }
                });
            };
        }
    }
}