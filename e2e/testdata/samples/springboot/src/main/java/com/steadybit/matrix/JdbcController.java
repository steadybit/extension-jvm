/*
 * Copyright 2026 steadybit GmbH. All rights reserved.
 */

package com.steadybit.matrix;

import org.springframework.jdbc.core.JdbcTemplate;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

@RestController
public class JdbcController {

    private final JdbcTemplate jdbcTemplate;

    public JdbcController(JdbcTemplate jdbcTemplate) {
        this.jdbcTemplate = jdbcTemplate;
        this.jdbcTemplate.execute("CREATE TABLE IF NOT EXISTS item (id INT PRIMARY KEY, name VARCHAR(64))");
        this.jdbcTemplate.update("MERGE INTO item KEY(id) VALUES (1, 'seed')");
    }

    @GetMapping("/jdbc")
    public String read() {
        Integer count = jdbcTemplate.queryForObject("SELECT count(*) FROM item", Integer.class);
        return "count=" + count;
    }
}
