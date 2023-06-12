/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.discovery.java.javaagent.handlers.datasource;

import com.steadybit.discovery.java.javaagent.handlers.instrumentation.ClassTransformationPlugin;
import com.steadybit.javaagent.instrumentation.Registration;
import com.steadybit.shaded.net.bytebuddy.agent.builder.AgentBuilder;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.*;

import javax.sql.DataSource;
import java.lang.instrument.Instrumentation;
import java.sql.Connection;
import java.util.List;

public class DataSourceScanner extends ClassTransformationPlugin {
    private final DataSourceConnections dataSourceConnections = new DataSourceConnections();

    public DataSourceScanner(Instrumentation instrumentation) {
        super(instrumentation);
    }

    @Override
    protected AgentBuilder doInstall(AgentBuilder agentBuilder) {
        return agentBuilder.type(hasSuperType(nameContains("javax.sql.DataSource")).and(not(isAbstract())))
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping().bind(Registration.class, this.getRegistration())).include(
                        CaptureDataSourceAdvice.class.getClassLoader())
                        .advice(named("getConnection").and(isPublic()).and(takesArguments(0)), CaptureDataSourceAdvice.class.getName()));
    }

    public List<DataSourceConnection> getDataSourceConnections() {
        return this.dataSourceConnections.getDatasourceConnections();
    }

    @Override
    public Object exec(int code, Object arg1, Object arg2) {
        if (code == 0) {
            this.dataSourceConnections.add((DataSource) arg1, (Connection) arg2);
        }
        return null;
    }

}
