/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import com.steadybit.attacks.javaagent.advice.Delay;
import com.steadybit.attacks.javaagent.advice.JdbcTemplateDelayAdvice;
import com.steadybit.attacks.javaagent.advice.JdbcUrl;
import com.steadybit.attacks.javaagent.advice.Jitter;
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

public class SpringJdbcTemplateDelayInstrumentation extends ClassTransformation {
    private static final String CLASSNAME_JDBC_TEMPLATE = "org.springframework.jdbc.core.JdbcTemplate";
    private final long delay;
    private final boolean delayJitter;
    private final String jdbcUrl;
    private ElementMatcher.Junction<MethodDescription> readMethodMatcher;
    private ElementMatcher.Junction<MethodDescription> writeMethodMatcher;

    public SpringJdbcTemplateDelayInstrumentation(Instrumentation instrumentation, JSONObject config) {
        super(instrumentation);
        this.writeMethodMatcher = none();
        this.readMethodMatcher = none();
        this.delay = config.optLong("delay", 500L);
        this.delayJitter = config.optBoolean("delayJitter", false);
        this.jdbcUrl = config.optString("jdbc-url", "*");
        this.initializeMatchers(config.optString("operations", "*"));
    }

    @Override
    protected AgentBuilder doInstall(AgentBuilder agentBuilder) {
        return agentBuilder.type(named(CLASSNAME_JDBC_TEMPLATE)) //
                .transform(new AgentBuilder.Transformer.ForAdvice(Advice.withCustomMapping() //
                        .bind(Delay.class, this.delay)//
                        .bind(Jitter.class, this.delayJitter)//
                        .bind(JdbcUrl.class, this.jdbcUrl))//
                        .include(JdbcTemplateDelayAdvice.class.getClassLoader()) //
                        .advice(this.readMethodMatcher, JdbcTemplateDelayAdvice.class.getName()) //
                        .advice(this.writeMethodMatcher, JdbcTemplateDelayAdvice.class.getName()));
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
