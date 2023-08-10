/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.instrumentation;

import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.util.Arrays;

import static org.assertj.core.api.Assertions.assertThat;

class InstrumentationPluginDispatcherTest {
    private final InstrumentationPlugin one = new InstrumentationPlugin();
    private final InstrumentationPlugin two = new InstrumentationPlugin();
    private final InstrumentationPlugin three = new InstrumentationPlugin();
    private final InstrumentationPlugin four = new InstrumentationPlugin();
    private final InstrumentationPlugin five = new InstrumentationPlugin();

    @BeforeEach
    void setUp() {
        InstrumentationPluginDispatcher.reset();
    }

    @Test
    void should_return_registered_plugin() {
        InstrumentationPluginDispatcher.register(this.one);
        InstrumentationPluginDispatcher.register(this.two);
        InstrumentationPluginDispatcher.register(this.three);

        assertThat(InstrumentationPluginDispatcher.find(this.one.getRegistration())).isSameAs(this.one);
        assertThat(InstrumentationPluginDispatcher.find(this.two.getRegistration())).isSameAs(this.two);
        assertThat(InstrumentationPluginDispatcher.find(this.three.getRegistration())).isSameAs(this.three);
    }

    @Test
    void should_return_noop() {
        assertThat(InstrumentationPluginDispatcher.find(0)).isSameAs(InstrumentationPlugin.NOOP);
    }

    @Test
    void should_deregister_plugin() {
        InstrumentationPluginDispatcher.register(this.one);
        int oldRegistration = this.one.getRegistration();
        InstrumentationPluginDispatcher.deregister(this.one);

        assertThat(InstrumentationPluginDispatcher.find(oldRegistration)).isSameAs(InstrumentationPlugin.NOOP);
        assertThat(this.one.getRegistration()).isEqualTo(-1);
    }

    @Test
    void should_fill_gaps() {
        InstrumentationPluginDispatcher.register(this.one);
        InstrumentationPluginDispatcher.register(this.two);
        InstrumentationPluginDispatcher.register(this.three);
        InstrumentationPluginDispatcher.deregister(this.two);
        InstrumentationPluginDispatcher.register(this.four);
        InstrumentationPluginDispatcher.register(this.five);

        assertThat(Arrays.copyOfRange(InstrumentationPluginDispatcher.plugins, 0, 5)).containsExactly(this.one, this.four, this.three, this.five,
                InstrumentationPlugin.NOOP);
    }
}
