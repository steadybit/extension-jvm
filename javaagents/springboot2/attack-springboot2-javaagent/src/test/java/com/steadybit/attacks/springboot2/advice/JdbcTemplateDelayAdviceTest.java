/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.springboot2.advice;

import static org.assertj.core.api.Assertions.assertThat;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import static org.mockito.Mockito.when;
import org.mockito.junit.jupiter.MockitoExtension;
import org.springframework.jdbc.core.JdbcTemplate;

import javax.sql.DataSource;
import java.sql.Connection;
import java.sql.DatabaseMetaData;

@ExtendWith(MockitoExtension.class)
class JdbcTemplateDelayAdviceTest {

    @Mock
    private DataSource dataSource;

    @Mock
    private Connection connection;

    @Mock
    private DatabaseMetaData databaseMetaData;

    @InjectMocks
    private JdbcTemplate jdbcTemplate;

    @Test
    void should_delay_200ms() {
        long delay = 200;
        boolean delayJitter = false;
        String jdbcUrl = "*";

        long startTime = System.currentTimeMillis();
        JdbcTemplateDelayAdvice.enter(delay, delayJitter, jdbcUrl, this.jdbcTemplate);

        long totalTime = System.currentTimeMillis() - startTime;
        assertThat(totalTime).isGreaterThanOrEqualTo(200);
    }

    @Test
    void should_delay_200ms_with_jitter() {
        long delay = 200;
        boolean delayJitter = true;
        String jdbcUrl = "*";

        long startTime = System.currentTimeMillis();
        JdbcTemplateDelayAdvice.enter(delay, delayJitter, jdbcUrl, this.jdbcTemplate);

        long totalTime = System.currentTimeMillis() - startTime;
        assertThat(totalTime).isBetween(140L, 260L);
    }

    @Test
    void should_delay_200ms_DbUrl() throws Exception {
        long delay = 200;
        boolean delayJitter = false;
        String jdbcUrl = "jdbc://testdb";

        when(this.databaseMetaData.getURL()).thenReturn(jdbcUrl);
        when(this.connection.getMetaData()).thenReturn(this.databaseMetaData);
        when(this.dataSource.getConnection()).thenReturn(this.connection);

        long startTime = System.currentTimeMillis();
        JdbcTemplateDelayAdvice.enter(delay, delayJitter, jdbcUrl, this.jdbcTemplate);

        long totalTime = System.currentTimeMillis() - startTime;
        assertThat(totalTime).isGreaterThanOrEqualTo(200);
    }

    @Test
    void should_not_delay_200ms_no_matching_dbUrl() throws Exception {
        long delay = 200;
        boolean delayJitter = false;
        String jdbcUrl = "jdbc://testdb";

        when(this.databaseMetaData.getURL()).thenReturn("jdbc://nomatch");
        when(this.connection.getMetaData()).thenReturn(this.databaseMetaData);
        when(this.dataSource.getConnection()).thenReturn(this.connection);

        long startTime = System.currentTimeMillis();
        JdbcTemplateDelayAdvice.enter(delay, delayJitter, jdbcUrl, this.jdbcTemplate);

        long totalTime = System.currentTimeMillis() - startTime;
        assertThat(totalTime).isLessThanOrEqualTo(30);
    }
}
