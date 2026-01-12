/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.beans;

import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;

import javax.management.InstanceNotFoundException;
import javax.management.MalformedObjectNameException;
import javax.management.MBeanServer;
import javax.management.ObjectName;
import java.lang.management.ManagementFactory;
import java.util.Map;

public class JmxBeanReader {
    private static final Logger log = RemoteAgentLogger.getLogger(JmxBeanReader.class);
    private final MBeanServer mBeanServer = ManagementFactory.getPlatformMBeanServer();
    private final ObjectName objectName;

    public JmxBeanReader() {
        try {
            this.objectName = new ObjectName("org.springframework.boot:type=Endpoint,name=Beans");
        } catch (MalformedObjectNameException e) {
            throw new RuntimeException("Could not create ObjectName for MBeans", e);
        }
    }

    public String getMainContextName() {
        try {
            Map<?, ?> result = (Map<?, ?>) this.mBeanServer.invoke(this.objectName, "beans", new Object[0], new String[0]);
            if (log.isTraceEnabled()) {
                log.trace("{}#beans() result: {}", this.objectName, result);
            }

            if (result == null || result.isEmpty()) {
                return null;
            }

            Map<?, ?> contexts = (Map<?, ?>) result.get("contexts");
            if (contexts == null || result.isEmpty()) {
                return null;
            }

            for (Map.Entry<?, ?> contextEntry : contexts.entrySet()) {
                Map<?, ?> context = (Map<?, ?>) contextEntry.getValue();
                String parentId = (String) context.get("parentId");

                String name = (String) contextEntry.getKey();
                if (parentId == null && !"bootstrap".equals(name)) {
                    return name;
                } else if ("bootstrap".equals(parentId)) {
                    return this.stripSuffix(name);
                }
            }
            return null;
        } catch (InstanceNotFoundException ex) {
            log.trace("Could not find main context: MBean {} not found", this.objectName);
            return null;
        } catch (Exception e) {
            log.debug("Could not find main context: " + e.getMessage());
            return null;
        }
    }

    private String stripSuffix(String s) {
        return s.replaceAll("-\\d+$", "");
    }

    public Boolean hasBeanOfType(Class<?> clazz) {
        try {
            Map<?, ?> result = (Map<?, ?>) this.mBeanServer.invoke(this.objectName, "beans", new Object[0], new String[0]);
            if (result == null || result.isEmpty()) {
                return null;
            }

            Map<?, ?> contexts = (Map<?, ?>) result.get("contexts");
            if (contexts == null || result.isEmpty()) {
                return null;
            }

            for (Map.Entry<?, ?> contextEntry : contexts.entrySet()) {
                Map<?, ?> context = (Map<?, ?>) contextEntry.getValue();
                Map<?, ?> beans = (Map<?, ?>) context.get("beans");

                if (beans == null) {
                    continue;
                }

                for (Map.Entry<?, ?> beanEntry : beans.entrySet()) {
                    Map<?, ?> bean = (Map<?, ?>) beanEntry.getValue();
                    String type = (String) bean.get("type");
                    if (type != null) {
                        try {
                            Class<?> beanClass = Class.forName(type);
                            if (clazz.isAssignableFrom(beanClass)) {
                                return true;
                            }
                        } catch (ClassNotFoundException e) {
                            //ignore
                        }
                    }
                }
            }
            return false;
        } catch (InstanceNotFoundException ex) {
            log.trace("Could not find bean of " + clazz + ": MBean org.springframework.boot:type=Endpoint,name=Beans not found");
            return null;
        } catch (Exception e) {
            log.debug("Could not find bean of " + clazz + ": " + e.getClass() + ": " + e.getMessage());
            return null;
        }
    }
}
