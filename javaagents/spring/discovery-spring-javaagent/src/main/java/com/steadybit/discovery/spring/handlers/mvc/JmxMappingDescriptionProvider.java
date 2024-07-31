/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.spring.handlers.mvc;

import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;
import org.json.JSONArray;
import org.json.JSONObject;

import javax.management.MBeanServer;
import javax.management.ObjectName;
import java.lang.management.ManagementFactory;
import java.util.Collection;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

public class JmxMappingDescriptionProvider {
    private static final Logger log = RemoteAgentLogger.getLogger(JmxMappingDescriptionProvider.class);
    private final MBeanServer mBeanServer = ManagementFactory.getPlatformMBeanServer();

    @SuppressWarnings("unchecked")
    public void describeMappings(JSONArray result) {
        try {
            ObjectName objectName = new ObjectName("org.springframework.boot:type=Endpoint,name=Mappings");
            Map<?, ?> applicationMappings = (Map<?, ?>) this.mBeanServer.invoke(objectName, "mappings", new Object[0], new String[0]);
            if (applicationMappings == null) {
                return;
            }

            Map<?, ?> contexts = (Map<?, ?>) applicationMappings.get("contexts");
            if (contexts == null) {
                return;
            }

            for (Map<?, ?> context : (Collection<Map<?, ?>>) contexts.values()) {
                Map<?, ?> contextMappings = (Map<?, ?>) context.get("mappings");
                if (contextMappings == null) {
                    continue;
                }

                Map<?, ?> dispatcherServletMappings = (Map<?, ?>) contextMappings.get("dispatcherServlets");
                if (dispatcherServletMappings != null) {
                    for (List<Map<?, ?>> mappings : (Collection<List<Map<?, ?>>>) dispatcherServletMappings.values()) {
                        for (Map<?, ?> mapping : mappings) {
                            this.describeMapping(mapping, result);
                        }
                    }
                }

                Map<?, ?> dispatcherHandlers = (Map<?, ?>) contextMappings.get("dispatcherHandlers");
                if (dispatcherHandlers != null) {
                    List<Map<?, ?>> webHandlers = (List<Map<?, ?>>) dispatcherHandlers.get("webHandler");
                    if (webHandlers != null) {
                        for (Map<?, ?> webHandler : webHandlers) {
                            this.describeMapping(webHandler, result);
                        }
                    }
                }
            }

        } catch (Exception e) {
            log.trace("Could not read spring mvc mappings from jmx", e);
        }
    }

    @SuppressWarnings("unchecked")
    private void describeMapping(Map<?, ?> mapping, JSONArray mappings) {
        Map<?, ?> details = (Map<?, ?>) mapping.get("details");
        if (details != null) {
            JSONObject json = new JSONObject();
            Map<?, ?> handlerMethod = (Map<?, ?>) details.get("handlerMethod");
            json.put("handlerClass", handlerMethod.get("className"));
            json.put("handlerName", handlerMethod.get("name"));
            json.put("handlerDescriptor", handlerMethod.get("descriptor"));

            Map<?, ?> requestMappingConditions = (Map<?, ?>) details.get("requestMappingConditions");
            putNotEmpty(json, "consumes", this.mediaTypeAsStringList((Collection<Map<?, ?>>) requestMappingConditions.get("consumes")));
            putNotEmpty(json, "headers", this.nameValueAsStringList((Collection<Map<?, ?>>) requestMappingConditions.get("headers")));
            putNotEmpty(json, "methods", requestMappingConditions.get("methods"));
            putNotEmpty(json, "params", this.nameValueAsStringList((Collection<Map<?, ?>>) requestMappingConditions.get("params")));
            putNotEmpty(json, "patterns", requestMappingConditions.get("patterns"));
            putNotEmpty(json, "produces", this.mediaTypeAsStringList((Collection<Map<?, ?>>) requestMappingConditions.get("produces")));
            mappings.put(json);
        }
    }

    private Collection<String> mediaTypeAsStringList(Collection<Map<?, ?>> expressions) {
        return expressions.stream().map(e -> {
            boolean negated = Boolean.TRUE.equals(e.get("negated"));
            return (negated ? "!" : "") + e.get("mediaType");
        }).collect(Collectors.toList());
    }

    private Collection<String> nameValueAsStringList(Collection<Map<?, ?>> expressions) {
        return expressions.stream().map(e -> {
            Object value = e.get("value");
            Object name = e.get("name");
            boolean negated = Boolean.TRUE.equals(e.get("negated"));
            if (value != null) {
                return name + (negated ? "!=" : "=") + value;
            } else {
                return (negated ? "!" : "") + name;
            }
        }).collect(Collectors.toList());
    }

    private static void putNotEmpty(JSONObject jsonObject, String key, Object value) {
        if (value != null && (!(value instanceof Collection) || !((Collection<?>) value).isEmpty())) {
            jsonObject.put(key, value);
        }
    }
}
