/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.spring.advice;

import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import org.springframework.http.client.reactive.ClientHttpResponse;
import reactor.core.publisher.Mono;

import java.net.URI;
import java.time.Duration;
import java.util.concurrent.ThreadLocalRandom;

public class WebClientDelayAdvice {
    @Advice.OnMethodExit
    static void exit(@Delay long delay, @Jitter boolean delayJitter, @HostAddress String hostAddress, @Advice.Argument(1) URI uri,
            @Advice.Return(readOnly = false) Mono<ClientHttpResponse> response) {

        if (!hostAddress.equals("*")) {
            String requestHostAddress = uri.getPort() == -1 ? uri.getHost() : uri.getHost() + ":" + uri.getPort();
            if (!requestHostAddress.equalsIgnoreCase(hostAddress)) {
                return;
            }
        }

        long millis;
        if (delayJitter) {
            double jitterValue = 1.3d - ThreadLocalRandom.current().nextDouble(0.6d);
            millis = Math.round(jitterValue * delay);
        } else {
            millis = delay;
        }

        response = Mono.delay(Duration.ofMillis(millis)).then(response);
    }
}
