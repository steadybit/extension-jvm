/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.java.javaagent.handlers.datasource;

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import javax.sql.DataSource;
import java.sql.Connection;
import java.sql.DatabaseMetaData;

import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;
import static org.mockito.Mockito.lenient;
import static org.mockito.Mockito.when;

@ExtendWith(MockitoExtension.class)
class DataSourceConnectionsTest {
    private final DataSourceConnections holder = new DataSourceConnections();

    @Mock
    private DataSource dataSourceMock;

    @Mock
    private Connection connectionMock;

    @Mock
    private DatabaseMetaData databaseMetaDataMock;


    @Test
    public void should_replace_pw_value_db2() throws Exception {
        lenient().when(this.dataSourceMock.getConnection()).thenReturn(this.connectionMock);
        when(this.connectionMock.getMetaData()).thenReturn(this.databaseMetaDataMock);
        when(this.databaseMetaDataMock.getURL()).thenReturn(
                "jdbc:db2://sysmvs1.stl.ibm.com:5021/SELECT1:user=dbadm;password=dbpassword;specialRegisters=CURRENT_PATH=SYSIBM,CURRENT CLIENT_USERID=test;");
        when(this.databaseMetaDataMock.getDatabaseProductName()).thenReturn("H2");

        this.holder.clear();
        this.holder.add(this.dataSourceMock, this.connectionMock);
        this.holder.getDatasourceConnections().forEach(con -> assertTrue(con.getJdbcUrl().contains("password=***")));

    }

    @Test
    public void should_not_replace_pw_db2() throws Exception {
        lenient().when(this.dataSourceMock.getConnection()).thenReturn(this.connectionMock);
        when(this.connectionMock.getMetaData()).thenReturn(this.databaseMetaDataMock);
        when(this.databaseMetaDataMock.getURL()).thenReturn(
                "jdbc:db2://sysmvs1.stl.ibm.com:5021/SELECT1:user=dbadm;specialRegisters=CURRENT_PATH=SYSIBM,CURRENT CLIENT_USERID=test;");
        when(this.databaseMetaDataMock.getDatabaseProductName()).thenReturn("H2");

        this.holder.clear();
        this.holder.add(this.dataSourceMock, this.connectionMock);
        this.holder.getDatasourceConnections().forEach(con -> assertFalse(con.getJdbcUrl().contains("password=***")));
    }

    @Test
    public void should_replace_pw_value_postgres() throws Exception {
        lenient().when(this.dataSourceMock.getConnection()).thenReturn(this.connectionMock);
        when(this.connectionMock.getMetaData()).thenReturn(this.databaseMetaDataMock);
        when(this.databaseMetaDataMock.getURL()).thenReturn(
                "jdbc:postgresql://localhost/test?user=fred&password=secret&ssl=true");
        when(this.databaseMetaDataMock.getDatabaseProductName()).thenReturn("H2");

        this.holder.clear();
        this.holder.add(this.dataSourceMock, this.connectionMock);
        this.holder.getDatasourceConnections().forEach(con -> assertTrue(con.getJdbcUrl().contains("password=***")));

    }

    @Test
    public void should_not_replace_pw_postgres() throws Exception {
        lenient().when(this.dataSourceMock.getConnection()).thenReturn(this.connectionMock);
        when(this.connectionMock.getMetaData()).thenReturn(this.databaseMetaDataMock);
        when(this.databaseMetaDataMock.getURL()).thenReturn(
                "jdbc:postgresql://localhost/test?user=fred&ssl=true");
        when(this.databaseMetaDataMock.getDatabaseProductName()).thenReturn("H2");

        this.holder.clear();
        this.holder.add(this.dataSourceMock, this.connectionMock);
        this.holder.getDatasourceConnections().forEach(con -> assertFalse(con.getJdbcUrl().contains("password=***")));
    }

    @Test
    public void should_replace_pw_value_oracle() throws Exception {
        lenient().when(this.dataSourceMock.getConnection()).thenReturn(this.connectionMock);
        when(this.connectionMock.getMetaData()).thenReturn(this.databaseMetaDataMock);
        when(this.databaseMetaDataMock.getURL()).thenReturn(
                "jdbc:oracle:oci:root/secretpassword@localhost:1521:testdb");
        when(this.databaseMetaDataMock.getDatabaseProductName()).thenReturn("H2");

        this.holder.clear();
        this.holder.add(this.dataSourceMock, this.connectionMock);
        this.holder.getDatasourceConnections().forEach(con -> assertTrue(con.getJdbcUrl().contains("/***@")));

    }

    @Test
    public void should_not_replace_pw_oracle() throws Exception {
        lenient().when(this.dataSourceMock.getConnection()).thenReturn(this.connectionMock);
        when(this.connectionMock.getMetaData()).thenReturn(this.databaseMetaDataMock);
        when(this.databaseMetaDataMock.getURL()).thenReturn(
                "jdbc:oracle:oci:root@localhost:1521:testdb");
        when(this.databaseMetaDataMock.getDatabaseProductName()).thenReturn("H2");

        this.holder.clear();
        this.holder.add(this.dataSourceMock, this.connectionMock);
        this.holder.getDatasourceConnections().forEach(con -> assertFalse(con.getJdbcUrl().contains("/***@")));
    }

    @Test
    public void should_replace_pw_value_sqlserver() throws Exception {
        lenient().when(this.dataSourceMock.getConnection()).thenReturn(this.connectionMock);
        when(this.connectionMock.getMetaData()).thenReturn(this.databaseMetaDataMock);
        when(this.databaseMetaDataMock.getURL()).thenReturn(
                "jdbc:sqlserver://localhost\\\\sqlexpress;user=sa;password=secret");
        when(this.databaseMetaDataMock.getDatabaseProductName()).thenReturn("H2");

        this.holder.clear();
        this.holder.add(this.dataSourceMock, this.connectionMock);
        this.holder.getDatasourceConnections().forEach(con -> assertTrue(con.getJdbcUrl().contains("password=***")));

    }

    @Test
    public void should_not_replace_pw_sqlserver() throws Exception {
        lenient().when(this.dataSourceMock.getConnection()).thenReturn(this.connectionMock);
        when(this.connectionMock.getMetaData()).thenReturn(this.databaseMetaDataMock);
        when(this.databaseMetaDataMock.getURL()).thenReturn(
                "jdbc:sqlserver://localhost\\\\sqlexpress;integratedSecurity=true");
        when(this.databaseMetaDataMock.getDatabaseProductName()).thenReturn("H2");

        this.holder.clear();
        this.holder.add(this.dataSourceMock, this.connectionMock);
        this.holder.getDatasourceConnections().forEach(con -> assertFalse(con.getJdbcUrl().contains("password=***")));
    }

    @Test
    public void should_replace_pw_value_mysql() throws Exception {
        lenient().when(this.dataSourceMock.getConnection()).thenReturn(this.connectionMock);
        when(this.connectionMock.getMetaData()).thenReturn(this.databaseMetaDataMock);
        when(this.databaseMetaDataMock.getURL()).thenReturn(
                "jdbc:mysql://localhost:3306/test?user=root&password=secret\n");
        when(this.databaseMetaDataMock.getDatabaseProductName()).thenReturn("H2");

        this.holder.clear();
        this.holder.add(this.dataSourceMock, this.connectionMock);
        this.holder.getDatasourceConnections().forEach(con -> assertTrue(con.getJdbcUrl().contains("password=***")));

    }

    @Test
    public void should_not_replace_pw_mysql() throws Exception {
        lenient().when(this.dataSourceMock.getConnection()).thenReturn(this.connectionMock);
        when(this.connectionMock.getMetaData()).thenReturn(this.databaseMetaDataMock);
        when(this.databaseMetaDataMock.getURL()).thenReturn(
                "jdbc:mysql://localhost:3306/test");
        when(this.databaseMetaDataMock.getDatabaseProductName()).thenReturn("H2");

        this.holder.clear();
        this.holder.add(this.dataSourceMock, this.connectionMock);
        this.holder.getDatasourceConnections().forEach(con -> assertFalse(con.getJdbcUrl().contains("password=***")));
    }


}