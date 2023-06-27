/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.java.javaagent.handlers.datasource;

public class DataSourceConnection {
    private final String jdbcUrl;
    private final String databaseType;

    public DataSourceConnection(String jdbcUrl, String databaseType) {
        this.jdbcUrl = jdbcUrl;
        this.databaseType = databaseType;
    }

    public String getJdbcUrl() {
        return this.jdbcUrl;
    }

    public String getDatabaseType() {
        return this.databaseType;
    }
}
