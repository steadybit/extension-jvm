/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.instrumentation;

import net.bytebuddy.description.type.TypeDescription;
import net.bytebuddy.matcher.ElementMatcher;
import org.junit.jupiter.api.Test;

import java.util.concurrent.atomic.AtomicBoolean;

import static net.bytebuddy.matcher.ElementMatchers.named;
import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatCode;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

class TransformationErrorListenerTest {
    private final TransformationErrorListener listener = new TransformationErrorListener();
    private final AtomicBoolean cleanedUp = new AtomicBoolean();

    @Test
    void should_throw_first_error() {
        this.listener.onError("com.example.First", null, null, true, new IllegalStateException("first"));
        this.listener.onError("com.example.Second", null, null, true, new IllegalStateException("second"));

        assertThatThrownBy(() -> this.listener.throwOnError(() -> this.cleanedUp.set(true)))
                .isInstanceOf(IllegalStateException.class)
                .hasMessageContaining("Failed to instrument com.example.First")
                .hasMessageContaining("first");
        assertThat(this.cleanedUp).isTrue();
    }

    @Test
    void should_not_throw_without_error() {
        assertThatCode(() -> this.listener.throwOnError(() -> this.cleanedUp.set(true))).doesNotThrowAnyException();
        assertThat(this.cleanedUp).isFalse();
    }

    @Test
    void should_ignore_errors_of_types_that_are_no_targets() {
        ElementMatcher<TypeDescription> targets = this.listener.targets(named(String.class.getName()));
        assertThat(targets.matches(TypeDescription.ForLoadedType.of(Integer.class))).isFalse();

        this.listener.onError(Integer.class.getName(), null, null, true, new IllegalStateException("matcher failed"));

        assertThatCode(() -> this.listener.throwOnError(() -> this.cleanedUp.set(true))).doesNotThrowAnyException();
    }

    @Test
    void should_throw_error_of_target() {
        ElementMatcher<TypeDescription> targets = this.listener.targets(named(String.class.getName()));
        assertThat(targets.matches(TypeDescription.ForLoadedType.of(String.class))).isTrue();

        this.listener.onError(Integer.class.getName(), null, null, true, new IllegalStateException("matcher failed"));
        this.listener.onError(String.class.getName(), null, null, true, new IllegalStateException("advice failed"));

        assertThatThrownBy(() -> this.listener.throwOnError(() -> this.cleanedUp.set(true)))
                .hasMessageContaining("Failed to instrument java.lang.String")
                .hasMessageContaining("advice failed");
    }
}
