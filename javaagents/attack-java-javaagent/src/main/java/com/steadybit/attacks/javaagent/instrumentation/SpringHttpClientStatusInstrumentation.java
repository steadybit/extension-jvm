/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import com.steadybit.attacks.javaagent.advice.RestTemplateHttpStatusAdvice;
import com.steadybit.attacks.javaagent.advice.WebClientHttpStatusAdvice;
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

public class SpringHttpClientStatusInstrumentation extends ClassTransformationPlugin {
    private static final int[] HTTP_4XX_CODES = {400, 401, 402, 403, 404, 405, 406, 407, 408, 409, 410, 411, 412, 413, 414, 415, 416, 417, 421, 422, 423, 424, 425, 426, 428, 429, 431, 451};
    private static final int[] HTTP_5XX_CODES = {500, 501, 502, 503, 504, 505, 506, 507, 508, 509, 510, 511};

    private final int errorRate;
    private final String[] httpMethods;
    private final String hostAddress;
    private final String urlPath;
    private final String[] failureCauses;
    private final ElementMatcher<MethodDescription> executeMethod = named("execute").and(takesNoArguments()).and(not(isAbstract()));
    private final ElementMatcher<MethodDescription> connectMethod = named("connect").and(takesArguments(3)).and(not(isAbstract()));

    public SpringHttpClientStatusInstrumentation(Instrumentation instrumentation, JSONObject config) {
        super(instrumentation);
        this.errorRate = config.optInt("erroneousCallRate", 100);
        this.hostAddress = config.optString("hostAddress", "*");
        this.urlPath = config.optString("urlPath", "*");
        this.httpMethods = this.getAllStringValues(config.optJSONArray("httpMethods"));
        this.failureCauses = this.getAllStringValues(config.optJSONArray("failureCauses"));
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
    protected AgentBuilder doInstall(AgentBuilder agentBuilder) {
        return agentBuilder //
                .type(hasSuperType(named("org.springframework.http.client.ClientHttpRequest")).and(declaresMethod(this.executeMethod)))
                .transform(new ClassInjectionTransformer(this.getClass().getClassLoader(), "com.steadybit.attacks.javaagent.advice.InjectedClientHttpResponse"))
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping() //
                        .bind(Registration.class, this.getRegistration())) //
                        .include(RestTemplateHttpStatusAdvice.class.getClassLoader()) //
                        .advice(this.executeMethod, RestTemplateHttpStatusAdvice.class.getName()))
                .type(hasSuperType(named("org.springframework.http.client.reactive.ClientHttpConnector")).and(declaresMethod(this.connectMethod)))
                .transform(new ClassInjectionTransformer(this.getClass().getClassLoader(),
                        "com.steadybit.attacks.javaagent.advice.InjectedReactiveClientHttpResponse"))
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping() //
                        .bind(Registration.class, this.getRegistration())) //
                        .include(WebClientHttpStatusAdvice.class.getClassLoader()) //
                        .advice(this.connectMethod, WebClientHttpStatusAdvice.class.getName()));
    }

    @Override
    public Object exec(int code, Object arg1, Object arg2, Object arg3, Object arg4) {
        if (code == 1) {
            return this.determineFailureScenario((String) arg1, (String) arg2, (int) arg3, (String) arg4);
        }
        return null;
    }

    private Integer determineFailureScenario(String httpMethod, String host, int port, String path) {
        if (!this.shouldAttack(httpMethod, host, port, path)) {
            return null;
        }

        if (this.errorRate < 100 && ThreadLocalRandom.current().nextInt(100) > this.errorRate) {
            return null;
        }

        return this.determineFailureStatusCode();
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

    private Integer determineFailureStatusCode() {
        if (this.failureCauses.length == 0) {
            return 500;
        }

        ThreadLocalRandom threadLocalRandom = ThreadLocalRandom.current();
        String failureCause = this.failureCauses[threadLocalRandom.nextInt(this.failureCauses.length)];

        if (HttpClientFailureCause.ERROR.equals(failureCause)) {
            return -2;
        } else if (HttpClientFailureCause.TIMEOUT.equals(failureCause)) {
            return -1;
        } else if (HttpClientFailureCause.HTTP_4XX.equals(failureCause)) {
            return HTTP_4XX_CODES[threadLocalRandom.nextInt(0, HTTP_4XX_CODES.length)];
        } else if (HttpClientFailureCause.HTTP_5XX.equals(failureCause)) {
            return HTTP_5XX_CODES[threadLocalRandom.nextInt(0, HTTP_5XX_CODES.length)];
        } else if (HttpClientFailureCause.HTTP_400.equals(failureCause)) {
            return 400;
        } else if (HttpClientFailureCause.HTTP_403.equals(failureCause)) {
            return 403;
        } else if (HttpClientFailureCause.HTTP_404.equals(failureCause)) {
            return 404;
        } else if (HttpClientFailureCause.HTTP_429.equals(failureCause)) {
            return 429;
        } else if (HttpClientFailureCause.HTTP_500.equals(failureCause)) {
            return 500;
        } else if (HttpClientFailureCause.HTTP_502.equals(failureCause)) {
            return 502;
        } else if (HttpClientFailureCause.HTTP_503.equals(failureCause)) {
            return 503;
        } else if (HttpClientFailureCause.HTTP_504.equals(failureCause)) {
            return 504;
        }

        return null;
    }
}
