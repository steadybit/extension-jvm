/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.mvc;

import org.json.JSONArray;
import org.json.JSONObject;
import org.springframework.asm.Type;
import org.springframework.boot.web.servlet.ServletRegistrationBean;
import org.springframework.context.ApplicationContext;
import org.springframework.util.ClassUtils;
import org.springframework.web.method.HandlerMethod;
import org.springframework.web.servlet.DispatcherServlet;
import org.springframework.web.servlet.HandlerMapping;
import org.springframework.web.servlet.mvc.condition.MediaTypeExpression;
import org.springframework.web.servlet.mvc.condition.NameValueExpression;
import org.springframework.web.servlet.mvc.condition.PathPatternsRequestCondition;
import org.springframework.web.servlet.mvc.condition.PatternsRequestCondition;
import org.springframework.web.servlet.mvc.method.RequestMappingInfo;
import org.springframework.web.servlet.mvc.method.RequestMappingInfoHandlerMapping;
import org.springframework.web.util.pattern.PathPattern;

import java.util.Collection;
import java.util.HashMap;
import java.util.HashSet;
import java.util.Map;
import java.util.Set;
import java.util.function.Supplier;
import java.util.stream.Collectors;

public class ServletMappingDescriptionProvider {

    private static final boolean dispatcherServletPresent = ClassUtils.isPresent("org.springframework.web.servlet.DispatcherServlet",
            ServletMappingDescriptionProvider.class.getClassLoader());
    private static final boolean servletRegistrationBeanPresent = ClassUtils.isPresent("org.springframework.boot.web.servlet.ServletRegistrationBean",
            ServletMappingDescriptionProvider.class.getClassLoader());
    private final Supplier<Collection<ApplicationContext>> applicationContextProvider;

    public ServletMappingDescriptionProvider(
            Supplier<Collection<ApplicationContext>> applicationContextProvider) {
        this.applicationContextProvider = applicationContextProvider;
    }

    public void describeMappings(JSONArray mappings) {
        if (ServletMappingDescriptionProvider.dispatcherServletPresent) {
            this.getDispatcherServlets().forEach((name, dispatcherServlet) -> this.describeMappings(dispatcherServlet, mappings));
        }
    }

    private void describeMappings(DispatcherServlet dispatcherServlet, JSONArray mappings) {
        if (dispatcherServlet.getHandlerMappings() != null) {
            for (HandlerMapping handlerMapping : dispatcherServlet.getHandlerMappings()) {
                this.describeMapping(handlerMapping, mappings);
            }
        }
    }

    private Map<String, DispatcherServlet> getDispatcherServlets() {
        Map<String, DispatcherServlet> dispatcherServlets = new HashMap<>();
        for (ApplicationContext applicationContext : this.applicationContextProvider.get()) {
            dispatcherServlets.putAll(applicationContext.getBeansOfType(DispatcherServlet.class));

            if (servletRegistrationBeanPresent) {
                for (ServletRegistrationBean<?> registrationBean : applicationContext.getBeansOfType(ServletRegistrationBean.class).values()) {
                    if (registrationBean.getServlet() instanceof DispatcherServlet) {
                        dispatcherServlets.put(registrationBean.getServletName(), (DispatcherServlet) registrationBean.getServlet());
                    }
                }
            }
        }
        return dispatcherServlets;
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

                PathPatternsRequestCondition pathPatternsCondition;
                try {
                    pathPatternsCondition = info.getPathPatternsCondition();
                } catch (NoSuchMethodError e) {
                    pathPatternsCondition = null;
                }

                boolean patternsConditionsAvailable = patternsCondition != null;
                boolean pathPatternsConditionsAvailable = pathPatternsCondition != null;
                if (patternsConditionsAvailable || pathPatternsConditionsAvailable) {
                    Set<String> patterns = new HashSet<>();
                    if (patternsConditionsAvailable) {
                        patterns.addAll(patternsCondition.getPatterns());
                    }
                    if (pathPatternsConditionsAvailable) {
                        for (PathPattern pathPattern : pathPatternsCondition.getPatterns()) {
                            patterns.add(pathPattern.getPatternString());
                        }
                    }
                    if (!patterns.isEmpty()) {
                        putNotEmpty(json, "patterns", patterns);
                    }
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
