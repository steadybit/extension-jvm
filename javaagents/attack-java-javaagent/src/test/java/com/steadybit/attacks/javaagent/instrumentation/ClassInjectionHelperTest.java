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

    public static class Injected {
    }
}
