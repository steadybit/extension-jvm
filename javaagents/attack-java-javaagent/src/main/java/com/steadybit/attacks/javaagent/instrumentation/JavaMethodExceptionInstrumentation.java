/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import com.steadybit.attacks.javaagent.advice.ErrorRate;
import com.steadybit.attacks.javaagent.advice.JavaMethodExceptionAdvice;
import com.steadybit.shaded.net.bytebuddy.agent.builder.AgentBuilder;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import com.steadybit.shaded.net.bytebuddy.description.method.MethodDescription;
import com.steadybit.shaded.net.bytebuddy.description.type.TypeDescription;
import com.steadybit.shaded.net.bytebuddy.matcher.ElementMatcher;
import org.json.JSONObject;

import java.lang.instrument.Instrumentation;

public class JavaMethodExceptionInstrumentation extends AbstractJavaMethodInstrumentation {
    private final int errorRate;

    public JavaMethodExceptionInstrumentation(Instrumentation instrumentation, JSONObject config) {
        super(instrumentation, config);
        this.errorRate = config.optInt("erroneousCallRate", 100);
    }

    @Override
    protected AgentBuilder doInstall(AgentBuilder agentBuilder, ElementMatcher.Junction<? super TypeDescription> typeMatcher,
                                     ElementMatcher.Junction<? super MethodDescription> methodMatcher) {

        return agentBuilder.type(typeMatcher) //
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping()//
                        .bind(ErrorRate.class, this.errorRate)) //
                        .include(JavaMethodExceptionAdvice.class.getClassLoader()) //
                        .advice(methodMatcher, JavaMethodExceptionAdvice.class.getName()));
    }
}
