/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment;

public enum CommandResult {
    OK(0),
    ERROR(1),
    UNKNOWN(-1);

    private static CommandResult[] values = values(); //prevent excessive defensive copies
    private final int rc;

    CommandResult(int rc) {
        this.rc = rc;
    }

    public static CommandResult of(int rc) {
        values = values();
        for (var value : values) {
            if (rc == value.rc) {
                return value;
            }
        }
        return UNKNOWN;
    }
}
