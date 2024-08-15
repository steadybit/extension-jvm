/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import com.steadybit.javaagent.log.Logger;
import com.steadybit.javaagent.log.RemoteAgentLogger;
import com.steadybit.shaded.net.bytebuddy.agent.builder.AgentBuilder;
import com.steadybit.shaded.net.bytebuddy.description.method.MethodDescription;
import com.steadybit.shaded.net.bytebuddy.description.type.TypeDescription;
import com.steadybit.shaded.net.bytebuddy.matcher.ElementMatcher;
import org.json.JSONObject;

import java.lang.instrument.Instrumentation;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.concurrent.atomic.AtomicBoolean;

import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.isDeclaredBy;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.named;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.none;

public abstract class AbstractJavaMethodInstrumentation extends ClassTransformation {

    private static final Logger log = RemoteAgentLogger.getLogger(AbstractJavaMethodInstrumentation.class);

    private ElementMatcher.Junction<? super TypeDescription> typeMatcher;
    private ElementMatcher.Junction<? super MethodDescription> methodMatcher;
    private final AtomicBoolean typeMatched = new AtomicBoolean(false);
    private final AtomicBoolean methodMatched = new AtomicBoolean(false);

    public AbstractJavaMethodInstrumentation(Instrumentation instrumentation, JSONObject config) {
        super(instrumentation);
        this.initializeMatchers(config);
    }

    @Override
    protected AgentBuilder doInstall(AgentBuilder agentBuilder) {
        return this.doInstall(agentBuilder, typeDefinitions -> {
            boolean matches = this.typeMatcher.matches(typeDefinitions);
            if (matches) {
                log.debug("Matched type: " + typeDefinitions);
                this.typeMatched.set(true);
            }
            return matches;
        }, methodDescription -> {
            boolean matches = this.methodMatcher.matches(methodDescription);
            if (matches) {
                log.debug("Matched method: " + methodDescription);
                this.methodMatched.set(true);
            }
            return matches;
        });
    }

    protected abstract AgentBuilder doInstall(AgentBuilder agentBuilder, ElementMatcher<? super TypeDescription> typeMatcher, ElementMatcher<? super MethodDescription> methodMatcher);

    private void initializeMatchers(JSONObject config) {
        this.typeMatcher = none();
        this.methodMatcher = none();

        if (config.has("methods")) {
            Map<String, List<String>> classWithMethods = new HashMap<>();

            for (Object methods : config.getJSONArray("methods")) {
                if (methods instanceof String) {
                    String[] tokens = ((String) methods).split("#", 2);
                    if (tokens.length > 1) {
                        classWithMethods.computeIfAbsent(tokens[0], k -> new ArrayList<>(5)).add(tokens[1]);
                    }
                }
            }

            for (Map.Entry<String, List<String>> entry : classWithMethods.entrySet()) {
                String className = entry.getKey();
                List<String> methodNames = entry.getValue();

                this.typeMatcher = this.typeMatcher.or(named(className));
                for (String methodName : methodNames) {
                    this.methodMatcher = this.methodMatcher.or(named(methodName).and(isDeclaredBy(named(className))));
                }
            }
        }
    }

    @Override
    protected AdviceApplied getAdviceApplied() {
        if (this.typeMatched.get() && this.methodMatched.get()) {
            return AdviceApplied.APPLIED;
        }
        return AdviceApplied.NOT_APPLIED;
    }
}
