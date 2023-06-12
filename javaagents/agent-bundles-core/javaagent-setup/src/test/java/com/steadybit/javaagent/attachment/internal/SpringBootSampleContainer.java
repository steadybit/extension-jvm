/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal;

import com.github.dockerjava.api.command.InspectContainerResponse;
import com.steadybit.docker.DockerClient;
import com.steadybit.javaagent.attachment.JavaVm;
import org.slf4j.LoggerFactory;
import org.testcontainers.containers.GenericContainer;
import org.testcontainers.containers.output.OutputFrame;
import org.testcontainers.containers.output.Slf4jLogConsumer;
import org.testcontainers.containers.startupcheck.OneShotStartupCheckStrategy;
import org.testcontainers.containers.wait.strategy.Wait;

import java.util.regex.Pattern;

public class SpringBootSampleContainer extends GenericContainer<SpringBootSampleContainer> {
    private static final Pattern PID_PATTERN = Pattern.compile("with PID (\\d+) ");
    private JavaVm javaVm;

    public SpringBootSampleContainer() {
        this("steadybit/spring-boot-sample:1.0.19");
    }

    public SpringBootSampleContainer(String dockerImageName) {
        super(dockerImageName);
        this.withEnv("STEADYBIT_LOG_JAVAAGENT_STDOUT", "true");
        this.withEnv("STEADYBIT_LOG_LEVEL", "DEBUG");
        this.withEnv("SPRING_APPLICATION_NAME", "spring-boot-sample");
        this.withExposedPorts(8080);
        this.waitingFor(Wait.forLogMessage(".*Started Application.*", 1));
    }

    public SpringBootSampleContainer useLogger(String name) {
        this.withLogConsumer(new Slf4jLogConsumer(LoggerFactory.getLogger(name)));
        return this;
    }

    @Override
    protected void containerIsStarted(InspectContainerResponse containerInfo) {
        var logs = this.getLogs(OutputFrame.OutputType.STDOUT);
        var m = PID_PATTERN.matcher(logs);
        int inContainerPid;
        if (m.find()) {
            inContainerPid = Integer.parseInt(m.group(1));
        } else {
            throw new IllegalStateException("Couldn't read pid from log!");
        }

        this.javaVm = new JavaVm(this.getJavaProcessHostPid());
        this.javaVm.setContainerId(DockerClient.PREFIX.add(this.getContainerId()));
        this.javaVm.setInContainerPid(inContainerPid);
        this.javaVm.setPath("/usr/lib/jvm/default-jvm/bin/java");
        this.javaVm.setDiscoveredVia("SpringBootSampleContainer");
    }

    private int getJavaProcessHostPid() {
        //as this is the init pid and we need the one of the java process we lookup the pid using an auxiliary container
        var initPid = this.getContainerInfo().getState().getPidLong();
        var container = new GenericContainer<>("alpine")
                .withCommand("pgrep", "-P", initPid.toString())
                .withStartupCheckStrategy(new OneShotStartupCheckStrategy())
                .withCreateContainerCmdModifier(cmd -> cmd.getHostConfig().withPidMode("host"));
        try {
            container.start();
            return Integer.parseInt(container.getLogs(OutputFrame.OutputType.STDOUT).split("\n")[0]);
        } finally {
            container.stop();
        }
    }

    public JavaVm getJavaVm() {
        return this.javaVm;
    }

    @Override
    protected void configure() {
        this.withCreateContainerCmdModifier(cmd -> cmd.getHostConfig().withInit(true));
    }
}
