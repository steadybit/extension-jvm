/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment;

public class JvmAttachmentException extends RuntimeException {
    public JvmAttachmentException(String message) {
        super(message);
    }

    public JvmAttachmentException(String message, Throwable cause) {
        super(message, cause);
    }
}
