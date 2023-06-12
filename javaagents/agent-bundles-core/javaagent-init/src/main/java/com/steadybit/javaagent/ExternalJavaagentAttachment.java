/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent;

import com.sun.jna.Native;
import com.sun.jna.Platform;
import com.sun.jna.platform.unix.LibCAPI;
import net.bytebuddy.agent.ByteBuddyAgent;

import java.io.File;
import java.net.URISyntaxException;
import java.util.Arrays;
import java.util.Map;
import java.util.stream.Collectors;

/**
 * Main class to do a self attachment of this agent to a VM with the given PID
 */
public class ExternalJavaagentAttachment {

    public static void main(String[] args) throws Exception {
        Map<String, String> arguments = parseCommandLineArguments(args);
        System.out.println("ExternalJavaagentAttachment called with " + arguments.toString());

        String agentJar = arguments.get("agentJar");
        String pid = arguments.get("pid");
        String hostpid = arguments.get("hostpid");
        String host = arguments.get("host");
        String port = arguments.get("port");
        String groupId = arguments.get("gid");
        String userId = arguments.get("uid");

        if (groupId != null && userId != null) {
            System.out.println("Switching to uid:gid " + userId + ":" + groupId);
            if (Platform.isLinux()) {
                switchUser(com.sun.jna.platform.linux.LibC.INSTANCE, Integer.parseInt(userId), Integer.parseInt(groupId));
            } else if (Platform.isMac()) {
                switchUser(com.sun.jna.platform.mac.SystemB.INSTANCE, Integer.parseInt(userId), Integer.parseInt(groupId));
            } else if (!Platform.isWindows()) {
                switchUser(com.sun.jna.platform.unix.LibC.INSTANCE, Integer.parseInt(userId), Integer.parseInt(groupId));
            }
        }

        String options;
        if (hostpid != null) {
            options = String.format("agentJar=%s,pid=%s,host=%s,port=%s", agentJar, hostpid, host, port);
            System.out.println("Attaching to JVM with PID " + pid + " and Host PID " + hostpid);
        } else {
            options = String.format("agentJar=%s,pid=%s,host=%s,port=%s", agentJar, pid, host, port);
            System.out.println("Attaching to JVM with PID " + pid);
        }
        File ownJar = getOwnJar();
        if (!ownJar.exists()) {
            throw new IllegalStateException("Could not find own jar file: " + ownJar.getAbsolutePath());
        }
        if (!new File(agentJar).exists()) {
            throw new IllegalStateException("Could not find agentJar file to load: " + ownJar.getAbsolutePath());
        }
        ByteBuddyAgent.attach(ownJar, pid, options);
    }

    private static void switchUser(LibCAPI api, int uid, int gid) {
        if (api.setgid(gid) != 0) {
            System.out.println("setgid(" + gid + ") failed. errno: " + Native.getLastError());
        }
        if (api.setegid(gid) != 0) {
            System.out.println("setegid(" + gid + ") failed. errno: " + Native.getLastError());
        }
        if (api.setuid(uid) != 0) {
            System.out.println("setuid(" + uid + ") failed. errno: " + Native.getLastError());
        }
        if (api.seteuid(uid) != 0) {
            System.out.println("seteuid(" + uid + ") failed. errno: " + Native.getLastError());
        }
    }

    private static File getOwnJar() throws URISyntaxException {
        return new File(ExternalJavaagentAttachment.class.getProtectionDomain().getCodeSource().getLocation().toURI());
    }

    private static Map<String, String> parseCommandLineArguments(String[] args) {
        return Arrays.stream(args).map(arg -> arg.split("=", 2)).collect(Collectors.toMap(arg -> arg[0], arg -> arg[1]));
    }
}
