/*
 * Copyright 2021 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment;

import java.util.Collection;
import java.util.Optional;

public interface JavaVms {
    Collection<JavaVm> getJavaVms();

    Optional<JavaVm> getJavaVm(int pid);

    void addListener(Listener l);

    void removeListener(Listener l);

    interface Listener {
        void addedJvm(JavaVm jvm);

        void removedJvm(JavaVm jvm);
    }
}
