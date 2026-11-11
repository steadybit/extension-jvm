/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import net.bytebuddy.agent.ByteBuddyAgent;
import org.junit.jupiter.api.Test;

import java.net.URL;
import java.net.URLClassLoader;
import java.util.Collections;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.assertThatThrownBy;

class ClassInjectionHelperTest {

    @Test
    void should_inject_class_without_unsafe() throws Exception {
        String className = Injected.class.getName();
        byte[] bytes = ClassInjectionHelper.readClass(Injected.class.getClassLoader(), className);

        try (URLClassLoader target = new URLClassLoader(new URL[0], null)) {
            assertThat(ClassInjectionHelper.hasClass(target, className)).isFalse();

            ClassInjectionHelper.injectUsingOpenedJavaLang(ByteBuddyAgent.install(), target, Collections.singletonMap(className, bytes));

            assertThat(target.loadClass(className).getClassLoader()).isSameAs(target);
        }
    }

    @Test
    void should_tolerate_class_that_is_already_injected() throws Exception {
        String className = Injected.class.getName();
        byte[] bytes = ClassInjectionHelper.readClass(Injected.class.getClassLoader(), className);

        try (URLClassLoader target = new URLClassLoader(new URL[0], null)) {
            ClassInjectionHelper.injectUsingOpenedJavaLang(ByteBuddyAgent.install(), target, Collections.singletonMap(className, bytes));
            Class<?> first = target.loadClass(className);

            ClassInjectionHelper.injectUsingOpenedJavaLang(ByteBuddyAgent.install(), target, Collections.singletonMap(className, bytes));

            assertThat(target.loadClass(className)).isSameAs(first);
        }
    }

    @Test
    void should_fail_for_invalid_class() throws Exception {
        try (URLClassLoader target = new URLClassLoader(new URL[0], null)) {
            assertThatThrownBy(() -> ClassInjectionHelper.injectUsingOpenedJavaLang(ByteBuddyAgent.install(), target,
                    Collections.singletonMap("com.example.Broken", new byte[]{1, 2, 3})))
                    .isInstanceOf(IllegalStateException.class)
                    .hasMessageContaining("com.example.Broken");
        }
    }

    public static class Injected {
    }
}
