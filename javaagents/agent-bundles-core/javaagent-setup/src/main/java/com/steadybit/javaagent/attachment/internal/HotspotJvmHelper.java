/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import com.steadybit.agent.system.SystemInfo;
import com.steadybit.javaagent.attachment.JavaVm;
import org.apache.commons.lang3.StringUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;
import sun.jvmstat.monitor.MonitorException;
import sun.jvmstat.monitor.StringMonitor;
import sun.jvmstat.perfdata.monitor.AbstractPerfDataBufferPrologue;
import sun.jvmstat.perfdata.monitor.PerfDataBufferImpl;
import sun.jvmstat.perfdata.monitor.protocol.local.PerfDataFile;
import sun.jvmstat.perfdata.monitor.v1_0.PerfDataBuffer;

import java.io.IOException;
import java.nio.Buffer;
import java.nio.ByteBuffer;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.nio.file.StandardOpenOption;
import java.util.*;

public class HotspotJvmHelper {
    private static final Logger log = LoggerFactory.getLogger(HotspotJvmHelper.class);

    public Set<Integer> getJvmPids() {
        return this.findJvmPids(this.allHSPerfDataDirs());
    }

    public Set<Integer> getJvmPids(Path rootFs) {
        return this.findJvmPids(this.allHSPerfDataDirs(rootFs));
    }

    public JavaVm getJvm(int hostPid) {
        return this.findJvm(hostPid, hostPid, this.allHSPerfDataDirs());
    }

    public JavaVm getJvmFromRoot(int pid, int hostPid, Path rootFs) {
        return this.findJvm(pid, hostPid, this.allHSPerfDataDirs(rootFs));
    }

    public JavaVm getJvmFromHsPerfDataDir(int pid, Path temp) {
        return this.findJvm(pid, pid, Collections.singletonList(temp));
    }

    private List<Path> allHSPerfDataDirs() {
        return this.allHSPerfDataDirs(null);
    }

    private List<Path> allHSPerfDataDirs(Path rootFs) {
        var perfDataTempDirs = this.getPerfDataTempDirs(rootFs);

        List<Path> hsPerfDataDirs = new ArrayList<>();
        for (var perfDataTempDir : perfDataTempDirs) {
            try (var perfDataDirs = Files.newDirectoryStream(perfDataTempDir,
                    entry -> entry.getFileName().toString().startsWith("hsperfdata_"))) {
                for (var perfDataDir : perfDataDirs) {
                    hsPerfDataDirs.add(perfDataDir);
                }
            } catch (IOException ex) {
                log.debug("Cannot access {}", perfDataTempDir);
            }
        }
        if (!hsPerfDataDirs.isEmpty()) {
            log.trace("Found PerfDataTempDirs in {}: {}", perfDataTempDirs, hsPerfDataDirs);
        }
        return hsPerfDataDirs;
    }

    private List<Path> getPerfDataTempDirs(Path rootFs) {
        List<Path> tmpDirs = new ArrayList<>();
        var dir = Paths.get(rootFs != null ? rootFs + PerfDataFile.getTempDirectory() : PerfDataFile.getTempDirectory());
        if (Files.isDirectory(dir)) {
            tmpDirs.add(dir);
        }
        return tmpDirs;
    }

    private Set<Integer> findJvmPids(List<Path> perfDataDirs) {
        Set<Integer> pids = new TreeSet<>();
        for (var hsPerfDataDir : perfDataDirs) {
            try (var hsPerfDataFiles = Files.newDirectoryStream(hsPerfDataDir,
                    entry -> StringUtils.isNumeric(entry.getFileName().toString()) && Files.isReadable(entry))) {
                for (var hsPerfDataFile : hsPerfDataFiles) {
                    log.trace("Found PID {} in {}.", hsPerfDataFile.getFileName(), hsPerfDataDir);
                    pids.add(Integer.valueOf(hsPerfDataFile.getFileName().toString()));
                }
            } catch (Exception ex) {
                log.debug("Cannot scan {}", hsPerfDataDir);
            }
        }
        return pids;
    }

    private JavaVm findJvm(int pid, int hostPid, List<Path> allHSPerfDataDirs) {
        for (var hsPerfDataDir : allHSPerfDataDirs) {
            if (Files.exists(hsPerfDataDir)) {
                try (var hsPerfDataFiles = Files.newDirectoryStream(hsPerfDataDir,
                        entry -> Integer.toString(pid).equals(entry.getFileName().toString()) && Files.isReadable(entry))) {
                    var hsPerfDataFilesIterator = hsPerfDataFiles.iterator();
                    if (!hsPerfDataFilesIterator.hasNext()) {
                        continue;
                    }
                    var hsPerfDataFile = hsPerfDataFilesIterator.next();
                    log.trace("Parsing PerfDataBuffer for pid {} in {}", hostPid, hsPerfDataFile);
                    return this.parsePerfDataBuffer(pid, hostPid, hsPerfDataFile);
                } catch (Exception ex) {
                    log.debug("Failed to parse {}", hsPerfDataDir, ex);
                }
            }
        }
        return null;
    }

    private JavaVm parsePerfDataBuffer(int pid, int hostPid, Path file) throws IOException, MonitorException {
        var buffer = this.getPerfDataBuffer(file);
        if (!this.isAttachable(buffer)) {
            log.trace("Jvm with pid {} discarded: not attachable", hostPid);
            return null;
        } else {
            // Extract vm data from PerfDataFile
            return this.buildFromPerfDataFile(pid, hostPid, buffer);
        }
    }

    private JavaVm buildFromPerfDataFile(int pid, int hostPid, PerfDataBufferImpl buffer) {
        var vm = new JavaVm(hostPid, "hsperfdata");
        var commandLine = this.getStringProperty(buffer, "sun.rt.javaCommand");
        vm.setCommandLine(commandLine);
        vm.setMainClass(this.getMainClass(commandLine));
        vm.setClasspath(this.getStringProperty(buffer, "java.property.java.class.path"));
        vm.setVmArgs(this.getStringProperty(buffer, "java.rt.vmArgs"));
        vm.setVmName(this.getStringProperty(buffer, "java.vm.name"));
        vm.setVmVendor(this.getStringProperty(buffer, "java.vm.vendor"));
        vm.setVmVersion(this.getStringProperty(buffer, "java.vm.version"));

        var process = SystemInfo.getOperatingSystem().getProcess(hostPid);
        if (process != null) {
            vm.setUserId(process.getUserID());
            vm.setGroupId(process.getGroupID());
            vm.setPath(process.getPath());
        }
        return vm;
    }

    /**
     * Returns the main class like JPS does. This is a rip-off from MonitoredVmUtil.
     */
    private String getMainClass(String commandLine) {
        if (StringUtils.isEmpty(commandLine)) {
            return null;
        }

        var cmdLine = commandLine;
        var firstSpace = cmdLine.indexOf(' ');
        if (firstSpace > 0) {
            cmdLine = cmdLine.substring(0, firstSpace);
        }
        /*
         * Can't use File.separator() here because the separator for the target
         * jvm may be different than the separator for the monitoring jvm.
         * And we also strip embedded module e.g. "module/MainClass"
         */
        var lastSlash = cmdLine.lastIndexOf("/");
        var lastBackslash = cmdLine.lastIndexOf("\\");
        var lastSeparator = Math.max(lastSlash, lastBackslash);
        if (lastSeparator > 0) {
            cmdLine = cmdLine.substring(lastSeparator + 1);
        }

        var lastPackageSeparator = cmdLine.lastIndexOf('.');
        if (lastPackageSeparator > 0) {
            var lastPart = cmdLine.substring(lastPackageSeparator + 1);
            /*
             * We could have a relative path "my.module" or
             * a module called "my.module" and a jar file called "my.jar" or
             * class named "jar" in package "my", e.g. "my.jar".
             * We can never be sure here, but we assume *.jar is a jar file
             */
            if (lastPart.equals("jar")) {
                return cmdLine; /* presumably a file name without path */
            }
            return lastPart; /* presumably a class name without package */
        }

        return cmdLine;
    }

    private String getStringProperty(PerfDataBufferImpl buffer, String propertyName) {
        try {
            var findByName = buffer.findByName(propertyName);
            if (findByName instanceof StringMonitor) {
                return ((StringMonitor) findByName).stringValue();
            }
        } catch (MonitorException ex) {
            log.error("Could not get property from perfdata", ex);
        }
        return "";
    }

    private boolean isAttachable(PerfDataBufferImpl buffer) {
        if (buffer != null) {
            try {
                var capabilities = (StringMonitor) buffer.findByName("sun.rt.jvmCapabilities");
                if (capabilities != null) {
                    var value = capabilities.stringValue();
                    return value.length() > 0 && value.charAt(0) == '1';
                }
            } catch (MonitorException ex) {
                log.error("Could not verify attach ability", ex);
            }
        }
        return false;
    }

    private PerfDataBufferImpl getPerfDataBuffer(Path path) throws IOException, MonitorException {
        try (var inChannel = Files.newByteChannel(path, StandardOpenOption.READ)) {
            var buffer = ByteBuffer.allocate((int) inChannel.size());
            inChannel.read(buffer);
            //noinspection RedundantCast
            ((Buffer) buffer).flip(); // we need this odd cast as there was an signature change with Java 9 in ByteBuffer
            if (!this.isAccessible(buffer)) {
                log.debug("hsperfdata file {} not accessible", path);
                return null;
            } else {
                var majorVersion = AbstractPerfDataBufferPrologue.getMajorVersion(buffer);
                switch (majorVersion) {
                    case 1:
                        return new PerfDataBuffer(buffer, -1);
                    case 2:
                        return new sun.jvmstat.perfdata.monitor.v2_0.PerfDataBuffer(buffer, -1);
                    default:
                        log.error("Incompatible JVM PerfData format {} in {}", majorVersion, path);
                        return null;
                }
            }
        }
    }

    private boolean isAccessible(ByteBuffer buffer) {
        //noinspection RedundantCast
        ((Buffer) buffer).position(7); // we need this odd cast as there was an signature change with Java 9 in ByteBuffer
        return buffer.get() != 0;
    }
}


