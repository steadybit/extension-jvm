/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent;

import java.io.OutputStream;

public interface CommandHandler {
    byte RC_OK = 0;
    byte RC_ERROR = 1;

    boolean canHandle(String command);

    void handle(String command, String argument, OutputStream os);
}
