/*
 * Copyright 2024 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.spring;

public interface Installable {

    default void install() {
    }

    default void reset() {
    }
}
