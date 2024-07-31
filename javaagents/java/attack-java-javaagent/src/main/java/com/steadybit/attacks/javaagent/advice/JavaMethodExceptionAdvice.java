/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.advice;

import com.steadybit.shaded.net.bytebuddy.asm.Advice;

import java.util.concurrent.ThreadLocalRandom;

public class JavaMethodExceptionAdvice {
    @Advice.OnMethodEnter
    static void enter(@ErrorRate int errorRate) {
        if (errorRate < 100) {
            if (ThreadLocalRandom.current().nextInt(100) < errorRate) {
                throw new RuntimeException("Exception injected by steadybit");
            }
        } else {
            throw new RuntimeException("Exception injected by steadybit");
        }
    }
}
