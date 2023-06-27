/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.java.javaagent.handlers.datasource;

import com.steadybit.javaagent.instrumentation.InstrumentationPluginDispatcher;
import com.steadybit.javaagent.instrumentation.Registration;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;

public class CaptureDataSourceAdvice {

    @Advice.OnMethodExit(suppress = Throwable.class)
    static void exit(@Registration int registration, @Advice.This Object dataSource, @Advice.Return Object connection) {
        InstrumentationPluginDispatcher.find(registration).exec(0, dataSource, connection);
    }
}
