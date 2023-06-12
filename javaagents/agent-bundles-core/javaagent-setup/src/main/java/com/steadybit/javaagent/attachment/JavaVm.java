/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment;

import org.apache.commons.lang3.StringUtils;

/**
 * All information about a discovered JVM.
 */
@lombok.Data
public class JavaVm {
    private final int pid;
    private String commandLine;
    private String mainClass;
    private String classpath;
    private String containerId;
    private Integer inContainerPid;
    private String vmVersion;
    private String vmVendor;
    private String vmName;
    private String vmArgs;
    private String userId;
    private String groupId;
    private String path;
    private String discoveredVia;

    public boolean isRunningInContainer() {
        return this.containerId != null;
    }

    public JavaVm(int pid) {
        this(pid, "unknown");
    }

    public JavaVm(int pid, String discoveredVia) {
        this.pid = pid;
        this.discoveredVia = discoveredVia;
    }

    public String toDebugString() {
        return "JavaVm{" +
                "pid=" + this.pid +
                ", discoveredVia=" + this.discoveredVia +
                ", commandLine='" + this.commandLine + '\'' +
                ", mainClass='" + this.mainClass + '\'' +
                ", classpath='" + this.classpath + '\'' +
                ", containerId='" + this.containerId + '\'' +
                ", inContainerPid=" + this.inContainerPid +
                ", vmVersion='" + this.vmVersion + '\'' +
                ", vmVendor='" + this.vmVendor + '\'' +
                ", vmName='" + this.vmName + '\'' +
                ", vmArgs='" + this.vmArgs + '\'' +
                ", userId='" + this.userId + '\'' +
                ", groupId='" + this.groupId + '\'' +
                ", path='" + this.path + '\'' +
                '}';
    }

    @Override
    public String toString() {
        return "JavaVm{" +
                "pid=" + this.pid +
                (this.inContainerPid != null ? "/" + this.inContainerPid : "") +
                (this.containerId != null ? ", containerId='" + StringUtils.left(this.containerId, 18) + "...'" : "") +
                (this.vmName != null ? ", vmName='" + this.vmName + '\'' : "") +
                (this.mainClass != null ? ", mainClass='" + this.mainClass + '\'' : "") +
                '}';
    }
}
