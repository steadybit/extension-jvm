/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.mvc;

import org.json.JSONArray;
import org.springframework.context.ApplicationContext;

import java.util.Collection;
import java.util.function.Supplier;

public class ApplicationContextMappingDescriptionProvider {

    private final ServletMappingDescriptionProvider servletMappingDescriptionProvider;
    private final ReactiveMappingDescriptionProvider reactiveMappingDescriptionProvider;

    public ApplicationContextMappingDescriptionProvider(Supplier<Collection<ApplicationContext>> applicationContextProvider) {
        this.servletMappingDescriptionProvider = new ServletMappingDescriptionProvider(applicationContextProvider);
        this.reactiveMappingDescriptionProvider = new ReactiveMappingDescriptionProvider(applicationContextProvider);
    }

    public void describeMappings(JSONArray mappings) {
        servletMappingDescriptionProvider.describeMappings(mappings);
        reactiveMappingDescriptionProvider.describeMappings(mappings);
    }

}