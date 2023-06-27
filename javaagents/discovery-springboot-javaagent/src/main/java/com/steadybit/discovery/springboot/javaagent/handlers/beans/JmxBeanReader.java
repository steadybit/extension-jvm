/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.beans;

import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;

import javax.management.InstanceNotFoundException;
import javax.management.MBeanServer;
import javax.management.ObjectName;
import java.lang.management.ManagementFactory;
import java.util.Map;

public class JmxBeanReader {
    private static final Logger log = RemoteAgentLogger.getLogger(JmxBeanReader.class);
    private final MBeanServer mBeanServer = ManagementFactory.getPlatformMBeanServer();

    public Boolean hasBeanOfType(Class<?> clazz) {
        try {
            ObjectName objectName = new ObjectName("org.springframework.boot:type=Endpoint,name=Beans");
            Map<?, ?> result = (Map<?, ?>) this.mBeanServer.invoke(objectName, "beans", new Object[0], new String[0]);
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
        }catch (Exception e) {
            log.debug("Could not find bean of " + clazz + ": " + e.getClass() +": " +e.getMessage());
            return null;
        }
    }
}
