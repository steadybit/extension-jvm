/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import java.net.InetAddress;

@lombok.Data
public class RemoteJvmConnection {
    public int pid;
    public InetAddress host;
    public Integer port;
}
