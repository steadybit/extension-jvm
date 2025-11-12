/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import com.steadybit.attacks.javaagent.advice.RestTemplateDelayAdvice;
import com.steadybit.attacks.javaagent.advice.WebClientDelayAdvice;
import com.steadybit.javaagent.instrumentation.Registration;
import com.steadybit.shaded.net.bytebuddy.agent.builder.AgentBuilder;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import com.steadybit.shaded.net.bytebuddy.description.method.MethodDescription;
import com.steadybit.shaded.net.bytebuddy.matcher.ElementMatcher;
import org.json.JSONArray;
import org.json.JSONObject;

import java.lang.instrument.Instrumentation;
import java.net.URI;
import java.util.List;
import java.util.concurrent.ThreadLocalRandom;
import java.util.stream.Collectors;

import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.declaresMethod;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.hasSuperType;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.isAbstract;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.named;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.not;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.takesArguments;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.takesNoArguments;

public class SpringHttpClientDelayInstrumentation extends ClassTransformationPlugin {
    private static final JSONArray EMPTY_ARRAY = new JSONArray();
    private final long delay;
    private final boolean delayJitter;
    private final ElementMatcher<MethodDescription> executeMethod = named("execute").and(takesNoArguments()).and(not(isAbstract()));
    private final ElementMatcher<MethodDescription> connectMethod = named("connect").and(takesArguments(3)).and(not(isAbstract()));
    private final List<String> httpMethods;
    private final String hostAdress;
    private final String urlPath;

    public SpringHttpClientDelayInstrumentation(Instrumentation instrumentation, JSONObject config) {
        super(instrumentation);
        this.delay = config.optLong("delay", 500L);
        this.delayJitter = config.optBoolean("delayJitter", false);
        this.httpMethods = config.optJSONArray("httpMethods", EMPTY_ARRAY).toList().stream().map(Object::toString).collect(Collectors.toList());
        this.hostAdress = config.optString("hostAddress", "*");
        this.urlPath = config.optString("urlPath", "/**");
    }

    @Override
    protected AgentBuilder doInstall(AgentBuilder agentBuilder) {
        return agentBuilder
                .type(hasSuperType(named("org.springframework.http.client.ClientHttpRequest")).and(declaresMethod(this.executeMethod)))
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping() //
                        .bind(Registration.class, this.getRegistration())) //
                        .include(RestTemplateDelayAdvice.class.getClassLoader()) //
                        .advice(this.executeMethod, RestTemplateDelayAdvice.class.getName()))
                .type(hasSuperType(named("org.springframework.http.client.reactive.ClientHttpConnector")).and(declaresMethod(this.connectMethod)))
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping() //
                        .bind(Registration.class, this.getRegistration())) //
                        .include(WebClientDelayAdvice.class.getClassLoader()) //
                        .advice(this.connectMethod, WebClientDelayAdvice.class.getName()));
    }

    @Override
    public Object exec(int code, Object arg1, Object arg2) {
        if (code == 2) {
            return this.determineDelay((String) arg1, (URI) arg2);
        }
        return null;
    }

    private Long determineDelay(String method, URI uri) {
        HttpMatcher m = new HttpMatcher(this.httpMethods, this.hostAdress, this.urlPath);
        if (!m.test(method, uri)) {
            return null;
        }

        if (this.delayJitter) {
            double jitterValue = 1.3d - ThreadLocalRandom.current().nextDouble(0.6d);
            return Math.round(jitterValue * this.delay);
        } else {
            return this.delay;
        }
    }
}
