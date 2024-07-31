/*
 * Copyright 2020 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import com.steadybit.shaded.net.bytebuddy.agent.builder.AgentBuilder;
import com.steadybit.shaded.net.bytebuddy.description.method.MethodDescription;
import com.steadybit.shaded.net.bytebuddy.description.type.TypeDescription;
import com.steadybit.shaded.net.bytebuddy.matcher.ElementMatcher;
import static com.steadybit.shaded.net.bytebuddy.matcher.ElementMatchers.*;
import org.json.JSONObject;

import java.lang.instrument.Instrumentation;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public abstract class AbstractJavaMethodInstrumentation extends ClassTransformation {
    private ElementMatcher.Junction<? super TypeDescription> typeMatcher;
    private ElementMatcher.Junction<? super MethodDescription> methodMatcher;

    public AbstractJavaMethodInstrumentation(Instrumentation instrumentation, JSONObject config) {
        super(instrumentation);
        this.initializeMatchers(config);
    }

    @Override
    protected AgentBuilder doInstall(AgentBuilder agentBuilder) {
        return this.doInstall(agentBuilder, this.typeMatcher, this.methodMatcher);
    }

    protected abstract AgentBuilder doInstall(AgentBuilder agentBuilder, ElementMatcher.Junction<? super TypeDescription> typeMatcher, ElementMatcher.Junction<? super MethodDescription> methodMatcher);

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
}
