/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */
package com.steadybit.attacks.javaagent.instrumentation;

import com.steadybit.attacks.javaagent.Installable;
import com.steadybit.javaagent.instrumentation.InstrumentationPlugin;
import com.steadybit.javaagent.instrumentation.InstrumentationPluginDispatcher;
import com.steadybit.javaagent.util.TempFileUtils;
import com.steadybit.shaded.net.bytebuddy.agent.builder.AgentBuilder;
import com.steadybit.shaded.net.bytebuddy.agent.builder.ResettableClassFileTransformer;

import java.io.File;
import java.io.IOException;
import java.lang.instrument.Instrumentation;

/**
 * TODO can we move this one to a central location?
 */
public abstract class ClassTransformationPlugin extends InstrumentationPlugin implements Installable {
    private static final File TEMP_DIR = getTempDir();
    private final Instrumentation instrumentation;
    private ResettableClassFileTransformer transformer;

    public ClassTransformationPlugin(Instrumentation instrumentation) {
        this.instrumentation = instrumentation;
    }

    @Override
    public AdviceApplied install() {
        InstrumentationPluginDispatcher.register(this);
        this.transformer = this.doInstall(this.createAgentBuilder()).installOn(this.instrumentation);
        return AdviceApplied.UNKNOWN;
    }

    protected AgentBuilder createAgentBuilder() {
        return new AgentBuilder.Default().disableClassFormatChanges()
                .with(new AgentBuilder.InjectionStrategy.UsingInstrumentation(this.instrumentation, TEMP_DIR))
                .with(AgentBuilder.RedefinitionStrategy.RETRANSFORMATION);
    }

    protected abstract AgentBuilder doInstall(AgentBuilder agentBuilder);

    @Override
    public void reset() {
        InstrumentationPluginDispatcher.deregister(this);
        if (this.transformer != null) {
            this.transformer.reset(this.instrumentation, AgentBuilder.RedefinitionStrategy.RETRANSFORMATION);
            this.transformer = null;
        }
    }

    private static File getTempDir() {
        try {
            return TempFileUtils.createTempDir("sb-agent");
        } catch (IOException e) {
            throw new RuntimeException(e);
        }
    }
}
