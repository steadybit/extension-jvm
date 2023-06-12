/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import com.steadybit.javaagent.attachment.JavaVm;
import static org.assertj.core.api.Assertions.assertThat;

import java.io.File;
import java.io.IOException;
import java.util.Scanner;

public class SpringBootSampleProcess {
    private final String perfdataFlag;
    private final String jarFile;
    private Process process;

    public SpringBootSampleProcess(boolean usePerfData) {
        this(usePerfData, "target/spring-boot-sample.jar");
    }

    public SpringBootSampleProcess(boolean usePerfData, String jarFile) {
        this.jarFile = jarFile;
        this.perfdataFlag = usePerfData ? "-XX:+UsePerfData" : "-XX:-UsePerfData";
    }

    public void start() {
        assertThat(new File(this.jarFile)).exists();
        var processBuilder = new ProcessBuilder("java", this.perfdataFlag, "-jar", "target/spring-boot-sample.jar", "--server.port=0");
        try {
            this.process = processBuilder.start();
        } catch (IOException e) {
            throw new IllegalStateException(e);
        }
        this.waitForStart();
    }

    public void stop() {
        if (this.process != null) {
            this.process.destroyForcibly();
        }
    }

    public JavaVm getJavaVm() {
        return new JavaVm(this.getPid());
    }

    private Integer getPid() {
        return Math.toIntExact(this.process.pid());
    }

    private void waitForStart() {
        try (var scanner = new Scanner(this.process.getInputStream())) {
            scanner.useDelimiter("\n");

            var deadline = System.currentTimeMillis() + 20_000L;
            while (scanner.hasNext() && System.currentTimeMillis() < deadline) {
                if (scanner.next().contains("Started Application")) {
                    return;
                }
            }
        }
    }
}