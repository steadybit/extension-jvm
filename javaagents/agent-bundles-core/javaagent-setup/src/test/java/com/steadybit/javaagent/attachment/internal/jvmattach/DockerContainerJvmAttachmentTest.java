/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal.jvmattach;

import com.steadybit.docker.DockerClient;
import com.steadybit.javaagent.attachment.internal.SpringBootSampleContainer;
import static org.assertj.core.api.Assertions.assertThat;
import org.junit.jupiter.api.Test;
import org.testcontainers.junit.jupiter.Container;
import org.testcontainers.junit.jupiter.Testcontainers;

@Testcontainers
class DockerContainerJvmAttachmentTest {
    @Container
    private static final SpringBootSampleContainer container = new SpringBootSampleContainer();
    @Container
    private static final SpringBootSampleContainer containerWithHostNetwork = new SpringBootSampleContainer()
            .withExposedPorts()
            .withNetworkMode("host");
    @Container
    private static final SpringBootSampleContainer containerWithContainerNetwork = new SpringBootSampleContainer()
            .withCommand("--server.port=8888")
            .dependsOn(container)
            .withExposedPorts()
            .withCreateContainerCmdModifier(cmd -> cmd.getHostConfig().withNetworkMode("container:" + container.getContainerId()));

    @Test
    void should_return_bridge_gateway_address_for_container() {
        var attachment = this.createDockerVmAttachment(container);

        var gateway = attachment.getContainerGatewayAddress();

        assertThat(gateway).isEqualTo("172.17.0.1");
    }

    @Test
    void should_return_bridge_gateway_address_for_container_with_container_network() {
        var attachment = this.createDockerVmAttachment(containerWithContainerNetwork);

        var gateway = attachment.getContainerGatewayAddress();

        assertThat(gateway).isEqualTo("172.17.0.1");
    }

    @Test
    void should_return_bridge_gateway_address_for_container_with_host_network() {
        var attachment = this.createDockerVmAttachment(containerWithHostNetwork);

        var gateway = attachment.getContainerGatewayAddress();

        assertThat(gateway).isEqualTo("127.0.0.1");
    }

    @Test
    void should_exec_command_with_overriden_env() {
        var attachment = this.createDockerVmAttachment(container);
        var result = attachment.exec(container.getContainerId(), "java", "-version");
        assertThat(result.getStdErr()).doesNotContain("-Dtool_options");
        assertThat(result.getStdErr()).doesNotContain("-Djava_options");
    }

    private DockerContainerJvmAttachment createDockerVmAttachment(SpringBootSampleContainer container) {
        return new DockerContainerJvmAttachment(container.getJavaVm(), new DockerClient()) {
            @Override
            protected String getAgentPublicAddress() {
                return null;
            }
        };
    }
}