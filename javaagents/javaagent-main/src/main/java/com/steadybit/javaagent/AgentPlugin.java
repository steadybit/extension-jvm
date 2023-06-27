/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent;

/**
 * Interface common to all agent plugins.
 *
 * <b>For an AgentPlugin to be loaded the manifest must have an entry `Agent-Plugin-Class` with it's classname.</b>
 * It also may have an `Agent-ClassLoader-Of` to specify other classloaders to be used.
 */
public interface AgentPlugin {

    default void start() {
    }

    default void destroy() {
    }
}
