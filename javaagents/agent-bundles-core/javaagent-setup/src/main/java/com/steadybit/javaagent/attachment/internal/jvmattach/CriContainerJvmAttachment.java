/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal.jvmattach;

import com.steadybit.containers.Container;
import com.steadybit.containers.ExecResult;
import com.steadybit.cri.CriContainerAdapter;
import com.steadybit.cri.CriClient;
import com.steadybit.javaagent.attachment.JavaVm;

import java.util.Optional;

public class CriContainerJvmAttachment extends ContainerJvmAttachment {
    private final CriClient criClient;

    public CriContainerJvmAttachment(JavaVm vm, CriClient criClient) {
        super(vm);
        this.criClient = criClient;
    }

    @Override
    protected ExecResult exec(String[] command) {
        return this.criClient.executeInContainer(this.vm.getContainerId(), command);
    }

    @Override
    protected Optional<Container> getContainer() {
        return this.criClient.getContainer(this.vm.getContainerId()).map(CriContainerAdapter::new);
    }

}
