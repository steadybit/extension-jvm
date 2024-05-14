/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.httpclient;

import com.steadybit.discovery.springboot.javaagent.instrumentation.ClassTransformationPlugin;
import com.steadybit.javaagent.instrumentation.Registration;
import com.steadybit.shaded.net.bytebuddy.agent.builder.AgentBuilder;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import com.steadybit.shaded.net.bytebuddy.description.method.MethodDescription;
import com.steadybit.shaded.net.bytebuddy.matcher.ElementMatcher;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.declaresMethod;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.hasSuperType;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.isAbstract;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.isPublic;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.isStatic;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.named;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.namedOneOf;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.not;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.takesArguments;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.takesNoArguments;
import okhttp3.OkHttpClient;
import org.apache.http.client.config.RequestConfig;
import org.apache.http.client.protocol.HttpClientContext;
import org.apache.http.protocol.HttpContext;
import org.springframework.http.client.ClientHttpRequest;
import org.springframework.http.client.reactive.ClientHttpResponse;
import reactor.core.publisher.Mono;
import rx.Observable;

import java.lang.instrument.Instrumentation;
import java.lang.reflect.Field;
import java.net.HttpURLConnection;
import java.net.URI;
import java.util.Collection;
import java.util.concurrent.ConcurrentHashMap;

public class HttpClientRequestScanner extends ClassTransformationPlugin {
    private final ConcurrentHashMap<String, HttpRequest> requests = new ConcurrentHashMap<>();
    private final ElementMatcher<MethodDescription> executeMethod = named("execute").and(takesNoArguments()).and(not(isAbstract()));
    private final ElementMatcher<MethodDescription> connectMethod = named("connect").and(takesArguments(3)).and(not(isAbstract()));
    private final ThreadLocal<Integer> circuitBreakerCounter = new ThreadLocal<>();
    private final ElementMatcher.Junction<MethodDescription> r4jCircuitBreakerMethod = namedOneOf("tryAcquirePermission", "acquirePermission",
            "releasePermission", "onError", "onSuccess").and(not(isAbstract().or(isStatic())).and(isPublic()));
    private final ElementMatcher<MethodDescription> runMethod = named("run").and(takesNoArguments()).and(not(isAbstract()));
    private final ElementMatcher<MethodDescription> toObservableMethod = named("toObservable").and(takesNoArguments());

    public HttpClientRequestScanner(Instrumentation instrumentation) {
        super(instrumentation);
    }

    @Override
    protected AgentBuilder doInstall(AgentBuilder agentBuilder) {
        return agentBuilder
                //For RestTemplate Requests
                .type(hasSuperType(named("org.springframework.http.client.ClientHttpRequest")).and(declaresMethod(this.executeMethod)))
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping()//
                        .bind(Registration.class, this.getRegistration()))//
                        .include(RestTemplateAdvice.class.getClassLoader()) //
                        .advice(this.executeMethod, RestTemplateAdvice.class.getName()))

                //For WebClient Requests
                .type(hasSuperType(named("org.springframework.http.client.reactive.ClientHttpConnector")).and(declaresMethod(this.connectMethod)))
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping() //
                        .bind(Registration.class, this.getRegistration()))//
                        .include(WebClientAdvice.class.getClassLoader()) //
                        .advice(this.connectMethod, WebClientAdvice.class.getName()))

                //For Resilience4j CircuitBreakers
                .type(hasSuperType(named("io.github.resilience4j.circuitbreaker.CircuitBreaker")).and(declaresMethod(this.r4jCircuitBreakerMethod)))
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping() //
                        .bind(Registration.class, this.getRegistration()))//
                        .include(Resilience4jCircuitBreakerAdvice.class.getClassLoader()) //
                        .advice(this.r4jCircuitBreakerMethod, Resilience4jCircuitBreakerAdvice.class.getName()))

                //For Hystrix CircuitBreakers
                .type(hasSuperType(named("com.netflix.hystrix.HystrixCommand")).and(declaresMethod(this.runMethod)))
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping() //
                        .bind(Registration.class, this.getRegistration()))//
                        .include(HystrixCircuitBreakerAdvice.class.getClassLoader()) //
                        .advice(this.runMethod, HystrixCircuitBreakerAdvice.class.getName()))

                //For Hystrix Observable CircuitBreakers
                .type(hasSuperType(named("com.netflix.hystrix.HystrixObservableCommand")))
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping() //
                        .bind(Registration.class, this.getRegistration()))//
                        .include(HystrixObservableCircuitBreakerAdvice.class.getClassLoader()) //
                        .advice(this.toObservableMethod, HystrixObservableCircuitBreakerAdvice.class.getName()));
    }

    public Collection<HttpRequest> getRequests() {
        return this.requests.values();
    }

    @Override
    public Object exec(int code) {
        if (code == 10) {
            this.incrementCircuitBreaker();
        } else if (code == 11) {
            this.decrementCircuitBreaker();
        }
        return null;
    }

    @Override
    public Object exec(int code, Object arg1) {
        if (code == 1) {
            this.scanRequest((ClientHttpRequest) arg1);
        } else if (code == 12) {
            return this.decorateHystrixObservable((Observable<?>) arg1);
        }
        return null;
    }

    @Override
    public Object exec(int code, Object arg1, Object arg2) {
        if (code == 0) {
            return this.decorateWebClientMono((URI) arg1, (Mono<ClientHttpResponse>) arg2);
        }
        return null;
    }

    private Observable<?> decorateHystrixObservable(Observable<?> obs) {
        return obs.doOnSubscribe(this::incrementCircuitBreaker).doOnTerminate(this::decrementCircuitBreaker);
    }

    private Mono<ClientHttpResponse> decorateWebClientMono(URI uri, Mono<ClientHttpResponse> mono) {
        return mono.doOnSubscribe(s -> this.scanAddress(uri, 0));
    }

    private void scanAddress(URI uri, int timeout) {
        if (uri != null && uri.getHost() != null) {
            String host = uri.getPort() == -1 ? uri.getHost() : uri.getHost() + ":" + uri.getPort();
            this.requests.put(host, new HttpRequest(host, uri.getScheme(), this.isCircuitBreakerActive(), timeout));
        }
    }

    private void scanRequest(ClientHttpRequest request) {
        if (request == null) {
            return;
        }
        this.scanAddress(request.getURI(), this.getTimeout(request));
    }

    private boolean isCircuitBreakerActive() {
        return this.circuitBreakerCounter.get() != null;
    }

    private void incrementCircuitBreaker() {
        Integer count = this.circuitBreakerCounter.get();
        this.circuitBreakerCounter.set(count == null ? 1 : count + 1);
    }

    private void decrementCircuitBreaker() {
        Integer count = this.circuitBreakerCounter.get();
        if (count != null) {
            if (count.equals(1))
                this.circuitBreakerCounter.remove();
            else
                this.circuitBreakerCounter.set(count - 1);
        }
    }

    private int getTimeout(ClientHttpRequest request) {
        Class<? extends ClientHttpRequest> requestClass = request.getClass();
        if (requestClass.getName().startsWith("org.springframework.http.client.Simple")) {
            // read socket timeout for simple jdk http client
            try {
                Field connectionField = requestClass.getDeclaredField("connection");
                connectionField.setAccessible(true);
                Object connection = connectionField.get(request);
                if (connection instanceof HttpURLConnection) {
                    return ((HttpURLConnection) connection).getReadTimeout();
                }
            } catch (Exception e) {
                //ignore
            }
        } else if (requestClass.getName().startsWith("org.springframework.http.client.HttpComponents")) {
            try {
                Field httpContextField = requestClass.getDeclaredField("httpContext");
                httpContextField.setAccessible(true);
                Object httpContext = httpContextField.get(request);
                if (httpContext instanceof HttpContext) {
                    Object requestConfig = ((HttpContext) httpContext).getAttribute(HttpClientContext.REQUEST_CONFIG);
                    if (requestConfig instanceof RequestConfig) {
                        return ((RequestConfig) requestConfig).getSocketTimeout();
                    } else {
                        return RequestConfig.DEFAULT.getSocketTimeout();
                    }
                }
            } catch (Exception e) {
                //ignore
            }
        } else if (requestClass.getName().startsWith("org.springframework.http.client.OkHttp3")) {
            try {
                // read socket timeout for OkHttp3
                Field clientField = requestClass.getDeclaredField("client");
                clientField.setAccessible(true);
                Object client = clientField.get(request);
                if (client instanceof OkHttpClient) {
                    return ((OkHttpClient) client).readTimeoutMillis();
                }
            } catch (Exception e) {
                //ignore
            }
        }
        return 0;
    }

}
