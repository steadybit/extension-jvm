/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.attacks.javaagent.instrumentation;

import com.steadybit.attacks.javaagent.Installable;
import com.steadybit.javaagent.util.TempFileUtils;
import com.steadybit.shaded.net.bytebuddy.agent.builder.AgentBuilder;
import com.steadybit.shaded.net.bytebuddy.agent.builder.ResettableClassFileTransformer;

import java.io.File;
import java.io.IOException;
import java.lang.instrument.Instrumentation;

public abstract class ClassTransformation implements Installable {
    private static final File TEMP_DIR = getTempDir();
    private final Instrumentation instrumentation;
    private ResettableClassFileTransformer transformer;

    protected ClassTransformation(Instrumentation instrumentation) {
        this.instrumentation = instrumentation;
    }

    @Override
    public AdviceApplied install() {
        this.transformer = this.doInstall(this.createAgentBuilder()).installOn(this.instrumentation);
        return this.getAdviceApplied();
    }

    protected AgentBuilder createAgentBuilder() {
        return new AgentBuilder.Default().disableClassFormatChanges()
                .with(new AgentBuilder.InjectionStrategy.UsingInstrumentation(this.instrumentation, TEMP_DIR))
                .with(AgentBuilder.RedefinitionStrategy.RETRANSFORMATION);
    }

    protected abstract AgentBuilder doInstall(AgentBuilder agentBuilder);

    @Override
    public void reset() {
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

    protected AdviceApplied getAdviceApplied() {
        return AdviceApplied.UNKNOWN;
    }
}
