/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.java.javaagent.handlers;

import com.steadybit.discovery.java.javaagent.handlers.datasource.DataSourceConnection;
import com.steadybit.javaagent.CommandHandler;
import org.json.JSONArray;
import org.json.JSONObject;

import java.io.OutputStream;
import java.io.OutputStreamWriter;
import java.io.PrintWriter;
import java.nio.charset.StandardCharsets;
import java.util.Collection;
import java.util.function.Supplier;

public class DataSourceCommandHandler implements CommandHandler {
    private static final char BYTE_ORDER_MARK = '\ufeff';
    private final Supplier<Collection<DataSourceConnection>> dataSourceConnectionProvider;

    public DataSourceCommandHandler(Supplier<Collection<DataSourceConnection>> dataSourceConnectionProvider) {
        this.dataSourceConnectionProvider = dataSourceConnectionProvider;
    }

    @Override
    public boolean canHandle(String command) {
        return command.equals("java-datasource-connection");
    }

    @Override
    public void handle(String command, String argument, OutputStream os) {
        JSONArray jdbcData = this.describeDataSourceConnections();
        PrintWriter writer = new PrintWriter(new OutputStreamWriter(os, StandardCharsets.UTF_8));
        writer.write(RC_OK);
        writer.write(BYTE_ORDER_MARK);
        jdbcData.write(writer);
        writer.flush();
    }

    private JSONArray describeDataSourceConnections() {
        JSONArray jdbcData = new JSONArray();
        this.dataSourceConnectionProvider.get().forEach(dataSourceConnection -> this.describeDataSourceConnection(dataSourceConnection, jdbcData));
        return jdbcData;
    }

    private void describeDataSourceConnection(DataSourceConnection dataSourceConnection, JSONArray jdbcData) {
        JSONObject json = new JSONObject();
        json.put("jdbcUrl", dataSourceConnection.getJdbcUrl());
        json.put("databaseType", dataSourceConnection.getDatabaseType());
        jdbcData.put(json);
    }
}
