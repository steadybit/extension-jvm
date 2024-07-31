/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.spring.handlers.env;

import org.springframework.context.ApplicationContext;

import java.util.Collection;
import java.util.function.Supplier;

public class ApplicationContextEnvironmentReader {
    private final Supplier<Collection<ApplicationContext>> applicationContextProvider;

    public ApplicationContextEnvironmentReader(Supplier<Collection<ApplicationContext>> applicationContextProvider) {
        this.applicationContextProvider = applicationContextProvider;
    }

    public String readPropertyValue(String name) {
        for (ApplicationContext applicationContext : this.applicationContextProvider.get()) {
            String value = applicationContext.getEnvironment().getProperty(name);
            if (value != null) {
                return value;
            }
        }
        return null;
    }
}
