/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import com.steadybit.agent.system.SystemInfo;
import org.apache.commons.io.FileUtils;
import static org.assertj.core.api.Assertions.assertThat;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.io.TempDir;
import org.mockito.MockedStatic;
import org.mockito.Mockito;
import static org.mockito.Mockito.mock;
import static org.mockito.Mockito.when;
import oshi.software.os.OSProcess;
import oshi.software.os.OperatingSystem;
import sun.jvmstat.perfdata.monitor.protocol.local.PerfDataFile;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.Collection;

class HotspotJvmHelperTest {
    private final HotspotJvmHelper hotspotJvmHelper = new HotspotJvmHelper();

    @Test
    void should_find_jvms(@TempDir Path root) throws IOException {
        this.mockHsPerfDataDir(root);

        Collection<Integer> jvmPids = this.hotspotJvmHelper.getJvmPids(root);
        assertThat(jvmPids).containsExactlyInAnyOrder(3043, 3180, 3998, 4001);
    }

    @Test
    void should_get_jvm(@TempDir Path root) throws IOException {
        this.mockHsPerfDataDir(root);
        try (var ms = this.mockProcess(3180)) {
            var jvm = this.hotspotJvmHelper.getJvmFromRoot(3180, 3180, root);
            assertThat(jvm.getPid()).isEqualTo(3180);
            assertThat(jvm.getMainClass()).isEqualTo("GradleDaemon");
            assertThat(jvm.getCommandLine()).isEqualTo("org.gradle.launcher.daemon.bootstrap.GradleDaemon 6.3");
            assertThat(jvm.getClasspath()).isEqualTo("/Users/jedmeier/.gradle/wrapper/dists/gradle-6.3-all/b4awcolw9l59x95tu1obfh9i8/gradle-6.3/lib/gradle-launcher-6.3.jar");
            assertThat(jvm.getContainerId()).isNull();
            assertThat(jvm.getUserId()).isEqualTo("1337");
            assertThat(jvm.getGroupId()).isEqualTo("1337");
        }
    }

    private MockedStatic<?> mockProcess(int pid) {
        var si = Mockito.mockStatic(SystemInfo.class);
        var os = mock(OperatingSystem.class);
        var process = mock(OSProcess.class);
        when(process.getUserID()).thenReturn("1337");
        when(process.getGroupID()).thenReturn("1337");
        when(os.getProcess(pid)).thenReturn(process);
        si.when(SystemInfo::getOperatingSystem).thenReturn(os);
        return si;
    }

    @Test
    void should_find_no_jvms(@TempDir Path root) {
        Collection<Integer> jvmPids = this.hotspotJvmHelper.getJvmPids(root);
        assertThat(jvmPids).isEmpty();
    }

    private void mockHsPerfDataDir(Path rootDir) throws IOException {
        var fakePerfDataDir = rootDir.resolve(PerfDataFile.getTempDirectory().substring(1)).resolve("hsperfdata_test");
        Files.createDirectories(fakePerfDataDir);

        var testData = Paths.get("src/test/resources/hsperfdata_test");
        FileUtils.copyDirectory(testData.toFile(), fakePerfDataDir.toFile());
    }
}