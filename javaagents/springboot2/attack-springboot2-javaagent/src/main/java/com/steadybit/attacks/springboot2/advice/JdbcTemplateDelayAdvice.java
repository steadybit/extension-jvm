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

public class JdbcTemplateDelayAdvice {
    @Advice.OnMethodEnter
    static void enter(@Delay long delay, @Jitter boolean delayJitter, @JdbcUrl String jdbcUrl, @Advice.This JdbcTemplate jdbcTemplate) {
        DataSource dataSource = jdbcTemplate.getDataSource();
        if (!jdbcUrl.equals("*") &&  dataSource != null) {
            try (Connection connection = dataSource.getConnection()) {
                if (!jdbcUrl.equalsIgnoreCase(connection.getMetaData().getURL())) {
                    return;
                }
            } catch (SQLException e) {
                //if we can't obtain the url we skip the attack
                return;
            }
        }

        long millis;
        if (delayJitter) {
            double jitterValue = 1.3d - ThreadLocalRandom.current().nextDouble(0.6d);
            millis = Math.round(jitterValue * delay);
        } else {
            millis = delay;
        }

        try {
            Thread.sleep(millis);
        } catch (InterruptedException e) {
            //ignore the interruption and restore interruption flag.
            Thread.currentThread().interrupt();
        }
    }
}
