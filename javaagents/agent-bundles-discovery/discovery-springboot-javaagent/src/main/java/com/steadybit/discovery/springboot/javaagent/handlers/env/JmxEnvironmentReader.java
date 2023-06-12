/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers.env;

import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;

import javax.management.InstanceNotFoundException;
import javax.management.MBeanServer;
import javax.management.ObjectName;
import java.lang.management.ManagementFactory;
import java.util.Map;

public class JmxEnvironmentReader {
    private static final Logger log = RemoteAgentLogger.getLogger(JmxEnvironmentReader.class);
    private final MBeanServer mBeanServer = ManagementFactory.getPlatformMBeanServer();

    public String readPropertyValue(String name) {
        try {
            ObjectName objectName = new ObjectName("org.springframework.boot:type=Endpoint,name=Env");
            Map<?, ?> env = (Map<?,?>) this.mBeanServer.invoke(objectName, "environmentEntry", new String[] { name }, new String[] { "java.lang.String" });
            if (env == null) {
                return null;
            }

            Map<?, ?> property = (Map<?,?>) env.get("property");
            if (property == null) {
                return null;
            }

            Object value = property.get("value");
            return value != null ? value.toString() : null;
        } catch (InstanceNotFoundException ex) {
            log.trace("Could not read " + name + " from spring environment. MBean org.springframework.boot:type=Endpoint,name=Env not found");
            return null;
        } catch (Exception e) {
            log.debug("Could not read " + name + " from spring environment: " + e.getClass() +": " +e.getMessage());
            return null;
        }
    }
}