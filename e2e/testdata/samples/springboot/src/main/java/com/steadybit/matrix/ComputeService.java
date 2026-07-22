/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package com.steadybit.matrix;

import org.springframework.stereotype.Service;

@Service
public class ComputeService {

    public String compute() {
        long sum = 0;
        for (int i = 0; i < 1000; i++) {
            sum += i;
        }
        return "computed-" + sum;
    }
}
