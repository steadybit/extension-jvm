/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent;

public class HttpAttackClientException extends RuntimeException {
    public HttpAttackClientException(String message) {
        super(message);
    }

    public HttpAttackClientException(Throwable cause) {
        super(cause);
    }

    public HttpAttackClientException(String message, Throwable cause) {
        super(message, cause);
    }
}
