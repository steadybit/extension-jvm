/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.spring;

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
