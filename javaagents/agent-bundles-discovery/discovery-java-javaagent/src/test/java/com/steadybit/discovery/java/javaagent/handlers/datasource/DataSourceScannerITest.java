/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.java.javaagent.handlers.datasource;

import net.bytebuddy.agent.ByteBuddyAgent;
import static org.assertj.core.api.Assertions.assertThat;
import org.hsqldb.jdbc.JDBCPool;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.lang.instrument.Instrumentation;
import java.sql.Connection;
import java.sql.SQLException;

class DataSourceScannerITest {
    private static final Instrumentation INSTRUMENTATION = ByteBuddyAgent.install();
    private DataSourceScanner scanner;

    @BeforeEach
    void setUp() {
        this.scanner = new DataSourceScanner(INSTRUMENTATION);
        this.scanner.install();
    }

    @AfterEach
    void tearDown() {
        this.scanner.reset();
    }

    @Test
    void should_capture_datasource() throws SQLException {
        JDBCPool pool = new JDBCPool();
        pool.setUrl("jdbc:hsqldb:mem:test");
        Connection connection = pool.getConnection();
        connection.close();
        pool.close(0);

        assertThat(this.scanner.getDataSourceConnections()).anySatisfy(dsConn -> {
            assertThat(dsConn.getJdbcUrl().equals("jdbc:hsqldb:mem:test"));
            assertThat(dsConn.getDatabaseType().equals("HSQLDB"));
        });
    }
}