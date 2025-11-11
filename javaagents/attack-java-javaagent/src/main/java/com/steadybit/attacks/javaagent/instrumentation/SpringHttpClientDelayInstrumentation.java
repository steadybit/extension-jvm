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
import java.util.Locale;
import java.util.concurrent.ThreadLocalRandom;

import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.declaresMethod;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.hasSuperType;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.isAbstract;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.named;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.not;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.takesArguments;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.takesNoArguments;

public class SpringHttpClientDelayInstrumentation extends ClassTransformationPlugin {
    private final long delay;
    private final boolean delayJitter;
    private final String[] httpMethods;
    private final String hostAddress;
    private final String urlPath;
    private final ElementMatcher<MethodDescription> executeMethod = named("execute").and(takesNoArguments()).and(not(isAbstract()));
    private final ElementMatcher<MethodDescription> connectMethod = named("connect").and(takesArguments(3)).and(not(isAbstract()));

    public SpringHttpClientDelayInstrumentation(Instrumentation instrumentation, JSONObject config) {
        super(instrumentation);
        this.delay = config.optLong("delay", 500L);
        this.delayJitter = config.optBoolean("delayJitter", false);
        this.hostAddress = config.optString("hostAddress", "*");
        this.urlPath = config.optString("urlPath", "*");
        this.httpMethods = this.getAllStringValues(config.optJSONArray("httpMethods"));
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

    private String[] getAllStringValues(JSONArray array) {
        if (array == null || array.isEmpty()) {
            return new String[]{};
        }

        String[] values = new String[array.length()];
        for (int i = 0; i < array.length(); i++) {
            values[i] = array.getString(i);
        }
        return values;
    }

    @Override
    public Object exec(int code, Object arg1, Object arg2, Object arg3, Object arg4) {
        if (code == 2) {
            return this.determineDelay((String) arg1, (String) arg2, (int) arg3, (String) arg4);
        }
        return null;
    }

    private Long determineDelay(String arg1, String arg2, int arg3, String arg4) {
        if (!this.shouldAttack(arg1, arg2, arg3, arg4)) {
            return null;
        }

        if (this.delayJitter) {
            double jitterValue = 1.3d - ThreadLocalRandom.current().nextDouble(0.6d);
            return Math.round(jitterValue * delay);
        } else {
            return this.delay;
        }
    }

    private boolean shouldAttack(String httpMethod, String host, int port, String path) {
        if (port != -1) {
            host = host + ":" + port;
        }

        if (!"*".equals(this.hostAddress) && !this.hostAddress.equalsIgnoreCase(host)) {
            return false;
        }

        if (!"*".equals(this.urlPath)
                && !"".equals(this.urlPath)
                && !this.urlPath.toLowerCase(Locale.ROOT).startsWith(path.toLowerCase(Locale.ROOT))) {
            return false;
        }

        if (this.httpMethods.length == 0) {
            return true;
        }

        for (String method : this.httpMethods) {
            if (method.equalsIgnoreCase(httpMethod)) {
                return true;
            }
        }

        return false;
    }

}
