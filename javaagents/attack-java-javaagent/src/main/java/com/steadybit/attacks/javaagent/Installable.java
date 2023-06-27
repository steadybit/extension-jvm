/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent;

public interface Installable {

    default void install() {
    }

    default void reset() {
    }
}
