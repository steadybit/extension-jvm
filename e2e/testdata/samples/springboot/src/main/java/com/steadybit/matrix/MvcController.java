/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package com.steadybit.matrix;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class MvcController {

    private final ComputeService computeService;

    public MvcController(ComputeService computeService) {
        this.computeService = computeService;
    }

    @GetMapping("/mvc")
    public String mvc() {
        return "ok:" + computeService.compute();
    }
}
