/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.spring.handlers.common;

import com.steadybit.javaagent.instrumentation.InstrumentationPluginDispatcher;
import com.steadybit.javaagent.instrumentation.Registration;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;

public class CaptureApplicationContextAdvice {

    @Advice.OnMethodEnter(suppress = Throwable.class)
    static void enter(@Registration int registration, @Advice.This Object applicationContext) {
        InstrumentationPluginDispatcher.find(registration).exec(0, applicationContext);
    }
}
