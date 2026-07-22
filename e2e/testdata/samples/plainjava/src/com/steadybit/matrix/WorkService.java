/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package com.steadybit.matrix;

public class WorkService {

    public String work() {
        long sum = 0;
        for (int i = 0; i < 1000; i++) {
            sum += i;
        }
        return "work-" + sum;
    }
}
