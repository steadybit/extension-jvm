/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.common;

import net.bytebuddy.agent.ByteBuddyAgent;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.context.annotation.AnnotationConfigApplicationContext;

import java.lang.instrument.Instrumentation;

import static org.assertj.core.api.Assertions.assertThat;

class ApplicationContextScannerTest {
    private static final Instrumentation INSTRUMENTATION = ByteBuddyAgent.install();
    private ApplicationContextScanner scanner;

    @BeforeEach
    void setUp() {
        this.scanner = new ApplicationContextScanner(INSTRUMENTATION);
        this.scanner.install();
    }

    @AfterEach
    void tearDown() {
        this.scanner.reset();
    }

    @Test
    void should_capture_application_context() {
        AnnotationConfigApplicationContext context = new AnnotationConfigApplicationContext();
        context.refresh();
        context.close();

        assertThat(this.scanner.getApplicationContexts()).contains(context);
    }
}