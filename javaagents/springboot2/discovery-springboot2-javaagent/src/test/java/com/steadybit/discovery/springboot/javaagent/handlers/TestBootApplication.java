/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.springboot.javaagent.handlers;

import org.springframework.boot.autoconfigure.SpringBootApplication;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@SpringBootApplication
@RestController
public class TestBootApplication {
    @GetMapping("/test")
    public String test() {
        return "Test";
    }

}