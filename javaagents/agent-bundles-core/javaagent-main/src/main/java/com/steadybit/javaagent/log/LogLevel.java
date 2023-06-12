/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.log;

public enum LogLevel {

    ERROR(0), WARN(1), INFO(2), DEBUG(3), TRACE(4);

    private final int level;

    LogLevel(int level) {
        this.level = level;
    }

    public int getLevel() {
        return this.level;
    }
}
