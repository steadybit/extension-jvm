/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal.jvmattach;

import java.io.File;
import java.util.Map;

public interface JvmAttachment {
    /**
     * attaches the given jars to the running VM
     *
     * @return true if successful, false if skipped
     */
    boolean attach(File agentJar, File initJar, int agentHttpPort);

    /**
     * copies the given files to the path inside the target container
     */
    void copyFiles(String dstPath, Map<String, File> files);

    /**
     * @return the host the agent is running on.
     */
    String getAgentHost();

}
