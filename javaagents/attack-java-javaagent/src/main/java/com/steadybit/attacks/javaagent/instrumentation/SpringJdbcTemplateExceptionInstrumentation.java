/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import com.steadybit.attacks.javaagent.advice.ErrorRate;
import com.steadybit.attacks.javaagent.advice.JdbcTemplateExceptionAdvice;
import com.steadybit.attacks.javaagent.advice.JdbcUrl;
import com.steadybit.shaded.net.bytebuddy.agent.builder.AgentBuilder;
import com.steadybit.shaded.net.bytebuddy.asm.Advice;
import com.steadybit.shaded.net.bytebuddy.description.method.MethodDescription;
import com.steadybit.shaded.net.bytebuddy.matcher.ElementMatcher;
import com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers;
import org.json.JSONObject;

import java.lang.instrument.Instrumentation;

import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.named;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.none;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.takesArgument;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.takesArguments;

public class SpringJdbcTemplateExceptionInstrumentation extends ClassTransformation {
    private static final String CLASSNAME_JDBC_TEMPLATE = "org.springframework.jdbc.core.JdbcTemplate";
    private final String jdbcUrl;
    private final int errorRate;
    private ElementMatcher.Junction<MethodDescription> readMethodMatcher;
    private ElementMatcher.Junction<MethodDescription> writeMethodMatcher;

    public SpringJdbcTemplateExceptionInstrumentation(Instrumentation instrumentation, JSONObject config) {
        super(instrumentation);
        this.writeMethodMatcher = none();
        this.readMethodMatcher = none();
        this.errorRate = config.optInt("erroneousCallRate", 100);
        this.jdbcUrl = config.optString("jdbc-url", "*");
        this.initializeMatchers(config.optString("operations", "*"));
    }

    @Override
    protected AgentBuilder doInstall(AgentBuilder agentBuilder) {
        return agentBuilder.type(named(CLASSNAME_JDBC_TEMPLATE)) //
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping() //
                        .bind(JdbcUrl.class, this.jdbcUrl.isEmpty() ? "*" : this.jdbcUrl)//
                        .bind(ErrorRate.class, this.errorRate))//
                        .include(JdbcTemplateExceptionAdvice.class.getClassLoader()) //
                        .advice(this.readMethodMatcher, JdbcTemplateExceptionAdvice.class.getName()) //
                        .advice(this.writeMethodMatcher, JdbcTemplateExceptionAdvice.class.getName()));
    }

    private void initializeMatchers(String operations) {
        // Any & Reads
        if (operations.equalsIgnoreCase("*") || operations.equalsIgnoreCase("r")) {
            this.readMethodMatcher = ElementMatchers.isPublic()
                    .and(named("query"))
                    .and(takesArgument(0, String.class))
                    .and(takesArgument(1, named("org.springframework.jdbc.core.ResultSetExtractor")));
        }

        // Any & Writes
        if (operations.equalsIgnoreCase("*") || operations.equalsIgnoreCase("w")) {
            ElementMatcher.Junction<MethodDescription> spring538andOlder = named("execute")
                    .and(takesArguments(2)
                            .and(takesArgument(0, named("org.springframework.jdbc.core.PreparedStatementCreator")))
                            .and(takesArgument(1, named("org.springframework.jdbc.core.PreparedStatementCallback")))
                    );
            ElementMatcher.Junction<MethodDescription> spring539andNewer = named("execute")
                    .and(
                            takesArguments(3)
                                    .and(takesArgument(0, named("org.springframework.jdbc.core.PreparedStatementCreator")))
                                    .and(takesArgument(1, named("org.springframework.jdbc.core.PreparedStatementCallback")))
                                    .and(takesArgument(2, boolean.class))

                    );
            this.writeMethodMatcher = spring539andNewer.or(spring538andOlder);
        }
    }
}
