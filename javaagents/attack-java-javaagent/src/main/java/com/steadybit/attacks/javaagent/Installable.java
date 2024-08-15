/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent;

public interface Installable {

    enum AdviceApplied {
        APPLIED, NOT_APPLIED, UNKNOWN
    }

    default AdviceApplied install() {
        return AdviceApplied.UNKNOWN;
    }

    default void reset() {
    }
}
