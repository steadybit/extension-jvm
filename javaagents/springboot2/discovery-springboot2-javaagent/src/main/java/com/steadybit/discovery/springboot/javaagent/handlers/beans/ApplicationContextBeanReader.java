/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.beans;

import org.springframework.context.ApplicationContext;

import java.util.Collection;
import java.util.function.Supplier;

public class ApplicationContextBeanReader {
    private final Supplier<Collection<ApplicationContext>> applicationContextProvider;

    public ApplicationContextBeanReader(Supplier<Collection<ApplicationContext>> applicationContextProvider) {
        this.applicationContextProvider = applicationContextProvider;
    }

    public boolean hasBeanOfType(Class<?> beanClass) {
        return this.applicationContextProvider.get().stream().anyMatch(c -> c.getBeanNamesForType(beanClass).length > 0);
    }
}
