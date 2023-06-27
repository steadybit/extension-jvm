/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import com.steadybit.attacks.javaagent.advice.*;
import com.steadybit.shaded.net.bytebuddy.agent.builder.AgentBuilder;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import com.steadybit.shaded.net.bytebuddy.description.method.MethodDescription;
import com.steadybit.shaded.net.bytebuddy.matcher.ElementMatcher;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.*;
import org.json.JSONObject;

import java.lang.instrument.Instrumentation;

public class SpringHttpClientDelayInstrumentation extends ClassTransformation {
    private final long delay;
    private final boolean delayJitter;
    private final String hostAddress;
    private final ElementMatcher<MethodDescription> executeMethod = named("execute").and(takesNoArguments()).and(not(isAbstract()));
    private final ElementMatcher<MethodDescription> connectMethod = named("connect").and(takesArguments(3)).and(not(isAbstract()));

    public SpringHttpClientDelayInstrumentation(Instrumentation instrumentation, JSONObject config) {
        super(instrumentation);
        this.delay = config.optLong("delay", 500L);
        this.delayJitter = config.optBoolean("delayJitter", false);
        this.hostAddress = config.optString("hostAddress", "*");
    }

    @Override
    protected AgentBuilder doInstall(AgentBuilder agentBuilder) {
        return agentBuilder
                .type(hasSuperType(named("org.springframework.http.client.ClientHttpRequest")).and(declaresMethod(this.executeMethod)))
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping() //
                        .bind(Delay.class, this.delay) //
                        .bind(Jitter.class, this.delayJitter) //
                        .bind(HostAddress.class, this.hostAddress)) //
                        .include(RestTemplateDelayAdvice.class.getClassLoader()) //
                        .advice(this.executeMethod, RestTemplateDelayAdvice.class.getName()))
                .type(hasSuperType(named("org.springframework.http.client.reactive.ClientHttpConnector")).and(declaresMethod(this.connectMethod)))
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping() //
                        .bind(Delay.class, this.delay) //
                        .bind(Jitter.class, this.delayJitter) //
                        .bind(HostAddress.class, this.hostAddress)) //
                        .include(WebClientDelayAdvice.class.getClassLoader()) //
                        .advice(this.connectMethod, WebClientDelayAdvice.class.getName()));
    }
}
