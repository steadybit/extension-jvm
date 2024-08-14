/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import com.steadybit.shaded.net.bytebuddy.agent.builder.AgentBuilder;
import com.steadybit.shaded.net.bytebuddy.description.type.TypeDescription;
import com.steadybit.shaded.net.bytebuddy.dynamic.DynamicType;
import com.steadybit.shaded.net.bytebuddy.dynamic.loading.ClassInjector;
import com.steadybit.shaded.net.bytebuddy.utility.JavaModule;

import java.security.ProtectionDomain;
import java.util.Arrays;
import java.util.List;
import java.util.stream.Collectors;

public class ClassInjectionTransformer implements AgentBuilder.Transformer {
    private final String[] classesToInject;
    private final ClassLoader sourceClassloader;

    public ClassInjectionTransformer(ClassLoader sourceClassloader, String... classesToInject) {
        this.classesToInject = classesToInject;
        this.sourceClassloader = sourceClassloader;
    }

    @Override
    public DynamicType.Builder<?> transform(DynamicType.Builder<?> builder, TypeDescription typeDescription, ClassLoader classLoader, JavaModule module,
                                            ProtectionDomain protectionDomain) {
        List<String> missingClasses = Arrays.stream(this.classesToInject)
                .filter(name -> !ClassInjectionHelper.hasClass(classLoader, name))
                .collect(Collectors.toList());
        if (!missingClasses.isEmpty()) {
            ClassInjector injector = ClassInjector.UsingUnsafe.isAvailable() ?
                    new ClassInjector.UsingUnsafe(classLoader) :
                    new ClassInjector.UsingReflection(classLoader);
            injector.injectRaw(ClassInjectionHelper.resolveClasses(missingClasses, this.sourceClassloader));
        }
        return builder;
    }
}
