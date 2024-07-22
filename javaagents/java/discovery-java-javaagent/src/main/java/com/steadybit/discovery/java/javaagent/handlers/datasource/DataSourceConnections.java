/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.java.javaagent.handlers.datasource;

import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;
import com.steadybit.javaagent.util.WeakConcurrentMap;

import javax.sql.DataSource;
import java.sql.Connection;
import java.sql.DatabaseMetaData;
import java.sql.SQLException;
import java.util.ArrayList;
import java.util.List;
import java.util.regex.Pattern;

public class DataSourceConnections {
    private final WeakConcurrentMap<DataSource, DataSourceConnection> connections = new WeakConcurrentMap<>(false);
    private final Pattern pwPattern = Pattern.compile("(password)=[^;&]*", Pattern.CASE_INSENSITIVE);
    private final Pattern oraclePwPattern = Pattern.compile("/[^@]+@");
    private static final Logger log = RemoteAgentLogger.getLogger(DataSourceConnections.class);

    public void add(DataSource dataSource, Connection connection) {
        if (!this.connections.containsKey(dataSource)) {
            try {
                DatabaseMetaData metaData = connection.getMetaData();
                DataSourceConnection conn = new DataSourceConnection(this.hidePasswordValueFromUrl(metaData.getURL()), metaData.getDatabaseProductName());
                log.trace("Observed new DataSourceConnection %s", conn.getJdbcUrl());
                this.connections.put(dataSource, conn);
            } catch (SQLException ex) {
                this.connections.put(dataSource, null);
            }
        }
    }

    private String hidePasswordValueFromUrl(String url) {
        String replacedOracle = this.oraclePwPattern.matcher(url).replaceAll("/***@");
        return this.pwPattern.matcher(replacedOracle).replaceAll("$1=***");
    }

    public List<DataSourceConnection> getDatasourceConnections() {
        this.connections.expungeStaleEntries();
        List<DataSourceConnection> result = new ArrayList<>();
        this.connections.iterator().forEachRemaining(dataSourceConnectionEntry -> {
            if (dataSourceConnectionEntry.getValue() != null)
                result.add(dataSourceConnectionEntry.getValue());
        });
        return result;
    }

    public void clear() {
        this.connections.clear();
    }

}
