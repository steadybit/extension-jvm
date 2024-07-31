/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import com.steadybit.attacks.javaagent.advice.Delay;
import com.steadybit.attacks.javaagent.advice.JavaMethodDelayAdvice;
import com.steadybit.attacks.javaagent.advice.Jitter;
import com.steadybit.shaded.net.bytebuddy.agent.builder.AgentBuilder;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import com.steadybit.shaded.net.bytebuddy.description.method.MethodDescription;
import com.steadybit.shaded.net.bytebuddy.description.type.TypeDescription;
import com.steadybit.shaded.net.bytebuddy.matcher.ElementMatcher;
import org.json.JSONObject;

import java.lang.instrument.Instrumentation;

public class JavaMethodDelayInstrumentation extends AbstractJavaMethodInstrumentation {
    private final long delay;
    private final boolean delayJitter;

    public JavaMethodDelayInstrumentation(Instrumentation instrumentation, JSONObject config) {
        super(instrumentation, config);
        this.delay = config.optLong("delay", 500L);
        this.delayJitter = config.optBoolean("delayJitter", false);
    }

    @Override
    protected AgentBuilder doInstall(AgentBuilder agentBuilder, ElementMatcher.Junction<? super TypeDescription> typeMatcher,
            ElementMatcher.Junction<? super MethodDescription> methodMatcher) {
        return agentBuilder.type(typeMatcher) //
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping() //
                        .bind(Delay.class, this.delay) //
                        .bind(Jitter.class, this.delayJitter)) //
                        .include(JavaMethodDelayAdvice.class.getClassLoader()) //
                        .advice(methodMatcher, JavaMethodDelayAdvice.class.getName()));
    }
}
