/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import com.steadybit.shaded.net.bytebuddy.agent.builder.AgentBuilder;
import com.steadybit.shaded.net.bytebuddy.description.type.TypeDescription;
import com.steadybit.shaded.net.bytebuddy.dynamic.DynamicType;
import com.steadybit.shaded.net.bytebuddy.utility.JavaModule;

import java.security.ProtectionDomain;
import java.util.Collections;

/**
 * Injects a single class — read from the agent classloader as a class-file resource — into the instrumented
 * target's classloader under its own name, unless the target already has it. Used to make the injected
 * client-http response classes available where the woven advice constructs them. Which variant is injected
 * (the Spring-5 or Spring-6 class) is decided by the instrumentation's classloader matcher, not here.
 */
public class NamedClassInjectionTransformer implements AgentBuilder.Transformer {
    private final ClassLoader agentClassLoader;
    private final String canonicalName;

    public NamedClassInjectionTransformer(ClassLoader agentClassLoader, String canonicalName) {
        this.agentClassLoader = agentClassLoader;
        this.canonicalName = canonicalName;
    }

    @Override
    public DynamicType.Builder<?> transform(DynamicType.Builder<?> builder, TypeDescription typeDescription, ClassLoader classLoader, JavaModule module,
                                            ProtectionDomain protectionDomain) {
        if (classLoader == null || ClassInjectionHelper.hasClass(classLoader, this.canonicalName)) {
            return builder;
        }
        byte[] bytes = ClassInjectionHelper.readClass(this.agentClassLoader, this.canonicalName);
        ClassInjectionHelper.inject(classLoader, Collections.singletonMap(this.canonicalName, bytes));
        return builder;
    }
}
