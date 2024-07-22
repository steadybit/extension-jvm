/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.advice;

import com.steadybit.shaded.net.bytebuddy.asm.Advice;

import java.util.concurrent.ThreadLocalRandom;

public class JavaMethodDelayAdvice {

    @Advice.OnMethodEnter
    static void enter(@Delay long delay, @Jitter boolean delayJitter) {
        long millis;
        if (delayJitter) {
            double jitterValue = 1.3d - ThreadLocalRandom.current().nextDouble(0.6d);
            millis = Math.round(jitterValue * delay);
        } else {
            millis = delay;
        }

        try {
            Thread.sleep(millis);
        } catch (InterruptedException e) {
            //ignore the interruption and restore interruption flag.
            Thread.currentThread().interrupt();
        }
    }
}
