/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.springboot2.advice;

import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import org.springframework.jdbc.core.JdbcTemplate;

import javax.sql.DataSource;
import java.sql.Connection;
import java.sql.SQLException;
import java.util.concurrent.ThreadLocalRandom;

public class JdbcTemplateExceptionAdvice {
    @Advice.OnMethodEnter
    static void enter(@ErrorRate int errorRate, @JdbcUrl String jdbcUrl, @Advice.This JdbcTemplate jdbcTemplate) {
        DataSource dataSource = jdbcTemplate.getDataSource();
        if (!jdbcUrl.equals("*") && dataSource != null) {
            try (Connection connection = dataSource.getConnection()) {
                if (!jdbcUrl.equalsIgnoreCase(connection.getMetaData().getURL())) {
                    return;
                }
            } catch (SQLException e) {
                //if we can't obtain the url we skip the attack
                return;
            }
        }

        if (errorRate >= 100 || ThreadLocalRandom.current().nextInt(100) < errorRate) {
            throw new RuntimeException("Exception injected by steadybit");
        }
    }
}
