/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.spring.handlers.mvc;

import org.json.JSONArray;
import org.json.JSONObject;
import org.springframework.asm.Type;
import org.springframework.context.ApplicationContext;
import org.springframework.util.ClassUtils;
import org.springframework.web.method.HandlerMethod;
import org.springframework.web.reactive.DispatcherHandler;
import org.springframework.web.reactive.HandlerMapping;
import org.springframework.web.reactive.result.condition.MediaTypeExpression;
import org.springframework.web.reactive.result.condition.NameValueExpression;
import org.springframework.web.reactive.result.condition.PatternsRequestCondition;
import org.springframework.web.reactive.result.method.RequestMappingInfo;
import org.springframework.web.reactive.result.method.RequestMappingInfoHandlerMapping;
import org.springframework.web.util.pattern.PathPattern;

import java.util.Collection;
import java.util.HashMap;
import java.util.Map;
import java.util.Set;
import java.util.function.Supplier;
import java.util.stream.Collectors;

public class ReactiveMappingDescriptionProvider {

    private static final boolean dispatcherHandlerPresent = ClassUtils.isPresent("org.springframework.web.reactive.DispatcherHandler",
            ApplicationContextMappingDescriptionProvider.class.getClassLoader());
    private final Supplier<Collection<ApplicationContext>> applicationContextProvider;

    public ReactiveMappingDescriptionProvider(
            Supplier<Collection<ApplicationContext>> applicationContextProvider) {
        this.applicationContextProvider = applicationContextProvider;
    }

    public void describeMappings(JSONArray mappings) {
        if (ReactiveMappingDescriptionProvider.dispatcherHandlerPresent) {
            this.getDispatcherHandler().forEach((name, dispatcherHandler) -> this.describeMappings(dispatcherHandler, mappings));
        }
    }

    private Map<String, DispatcherHandler> getDispatcherHandler() {
        Map<String, DispatcherHandler> dispatcherHandler = new HashMap<>();
        for (ApplicationContext applicationContext : this.applicationContextProvider.get()) {
            dispatcherHandler.putAll(applicationContext.getBeansOfType(DispatcherHandler.class));
        }
        return dispatcherHandler;
    }

    private void describeMappings(DispatcherHandler dispatcherHandler, JSONArray mappings) {
        if (dispatcherHandler.getHandlerMappings() != null) {
            for (HandlerMapping handlerMapping : dispatcherHandler.getHandlerMappings()) {
                this.describeMapping(handlerMapping, mappings);
            }
        }
    }

    private void describeMapping(HandlerMapping handlerMapping, JSONArray mappings) {
        if (handlerMapping instanceof RequestMappingInfoHandlerMapping) {
            RequestMappingInfoHandlerMapping mapping = (RequestMappingInfoHandlerMapping) handlerMapping;

            for (Map.Entry<RequestMappingInfo, HandlerMethod> entry : mapping.getHandlerMethods().entrySet()) {
                RequestMappingInfo info = entry.getKey();
                HandlerMethod handler = entry.getValue();

                JSONObject json = new JSONObject();
                putNotEmpty(json, "consumes", this.mediaTypeAsStringList(info.getConsumesCondition().getExpressions()));
                putNotEmpty(json, "headers", this.nameValueAsStringList(info.getHeadersCondition().getExpressions()));
                putNotEmpty(json, "methods", info.getMethodsCondition().getMethods());
                putNotEmpty(json, "params", this.nameValueAsStringList(info.getParamsCondition().getExpressions()));
                PatternsRequestCondition patternsCondition = info.getPatternsCondition();
                if (patternsCondition != null) {
                    putNotEmpty(json, "patterns",
                            patternsCondition.getPatterns().stream().map(PathPattern::getPatternString).collect(Collectors.toList()));
                }
                putNotEmpty(json, "produces", this.mediaTypeAsStringList(info.getProducesCondition().getExpressions()));
                json.put("handlerClass", handler.getMethod().getDeclaringClass().getName());
                json.put("handlerName", handler.getMethod().getName());
                json.put("handlerDescriptor", Type.getMethodDescriptor(handler.getMethod()));
                mappings.put(json);
            }
        }
    }

    private Collection<String> mediaTypeAsStringList(Set<MediaTypeExpression> expressions) {
        return expressions.stream().map(Object::toString).collect(Collectors.toList());
    }

    private Collection<String> nameValueAsStringList(Set<NameValueExpression<String>> expressions) {
        return expressions.stream().map(e -> {
            if (e.getValue() != null) {
                return e.getName() + (e.isNegated() ? "!=" : "=") + e.getValue();
            } else {
                return (e.isNegated() ? "!" : "") + e.getName();
            }
        }).collect(Collectors.toList());
    }

    private static void putNotEmpty(JSONObject jsonObject, String key, Collection<?> value) {
        if (value != null && !value.isEmpty()) {
            jsonObject.put(key, value);
        }
    }
}
