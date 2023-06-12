/*
 * Copyright 2022 steadybit GmbH. All rights reserved.
 */

package com.steadybit.javaagent.attachment.internal.jvmattach;

import com.steadybit.agent.commons.ProcFs;
import com.steadybit.containers.Container;
import com.steadybit.containers.ExecResult;
import com.steadybit.docker.DockerClient;
import com.steadybit.docker.DockerContainerAdapter;
import com.steadybit.docker.model.ContainerNetwork;
import com.steadybit.javaagent.attachment.JavaVm;
import com.steadybit.utils.ContainerNetworkUtil;
import com.sun.jna.Platform;

import java.io.File;
import java.io.IOException;
import java.util.ArrayList;
import java.util.Collection;
import java.util.Collections;
import java.util.HashMap;
import java.util.List;
import java.util.Map;
import java.util.Objects;
import java.util.Optional;

public class DockerContainerJvmAttachment extends ContainerJvmAttachment {
    private static final String LOOPBACK_ADDRESS = "127.0.0.1";
    private static final Map<String, String> ENV_OVERRIDES = getEnvOverrides();
    private final DockerClient dockerClient;

    public DockerContainerJvmAttachment(JavaVm vm, DockerClient dockerClient) {
        super(vm);
        this.dockerClient = dockerClient;
    }

    @Override
    protected Optional<Container> getContainer() {
        return this.dockerClient.getContainer(this.vm.getContainerId()).map(DockerContainerAdapter::new);
    }

    @Override
    protected ExecResult exec(String... command) {
        return this.dockerClient.executeInContainer(this.vm.getContainerId(), ENV_OVERRIDES, command);
    }

    @Override
    public void copyFiles(String dstPath, Map<String, File> files) {
        this.dockerClient.copyToContainer(this.vm.getContainerId(), dstPath, files);
    }

    @Override
    public String getAgentHost() {
        if (Platform.isMac() || Platform.isWindows()) {
            this.log.trace("Using host.docker.internal for container {} agent callback", this.vm.getContainerId());
            return "host.docker.internal";
        }

        var gateway = this.getContainerBridgeGatewayAddress();
        if (gateway != null) {
            return gateway;
        }

        gateway = this.getContainerGatewayAddress();
        if (gateway != null) {
            return gateway;
        }

        this.log.trace("Using loopback for container {} agent callback", this.vm.getContainerId());
        return LOOPBACK_ADDRESS;
    }

    String getContainerBridgeGatewayAddress() {
        var gateway = this.getContainerGatewayFromDockerInspect(this.vm.getContainerId(), true);
        if (gateway != null) {
            this.log.trace("Using bridge from inspect {} for container {} agent callback", gateway, this.vm.getContainerId());
            return gateway;
        }

        gateway = this.getContainerGatewayFromProc(this.vm.getPid(), true);
        if (gateway != null) {
            this.log.trace("Using bridge from proc {} for container {} agent callback", gateway, this.vm.getContainerId());
            return gateway;
        }

        gateway = ContainerNetworkUtil.findBridgeTo(this.getContainerAdresses());
        if (gateway != null) {
            this.log.trace("Using bridge to container address {} for container {} agent callback", gateway, this.vm.getContainerId());
            return gateway;
        }

        gateway = this.getAgentPublicAddress();
        if (this.isBridge(gateway)) {
            this.log.trace("Using container address bridge {} for container {} agent callback", gateway, this.vm.getContainerId());
            return gateway;
        }
        return null;
    }

    String getContainerGatewayAddress() {
        var gateway = this.getAgentPublicAddress();
        if (gateway != null) {
            this.log.trace("Using non-bridge container address {} for container {} agent callback", gateway, this.vm.getContainerId());
            return gateway;
        }

        gateway = this.getContainerGatewayFromDockerInspect(this.vm.getContainerId(), false);
        if (gateway != null) {
            this.log.trace("Using non-bridge {} for container {} agent callback", gateway, this.vm.getContainerId());
            return gateway;
        }

        gateway = this.getContainerGatewayFromProc(this.vm.getPid(), false);
        if (gateway != null) {
            this.log.trace("Using non-bridge {} from proc for container {} agent callback", gateway, this.vm.getContainerId());
            return gateway;
        }

        return null;
    }

    protected String getAgentPublicAddress() {
        return ContainerNetworkUtil.getAgentPublicAddress();
    }

    private String getContainerGatewayFromProc(int pid, boolean mustBeBridge) {
        try {
            var gateway = ProcFs.ROOT.readGatewayFromRoute(pid);
            return (mustBeBridge && !this.isBridge(gateway)) ? null : gateway;
        } catch (IOException ex) {
            this.log.debug("Could not read gateway from proc for {}", pid);
            return null;
        }
    }

    private String getContainerGatewayFromDockerInspect(String containerId, boolean mustBeBridge) {
        var inspect = this.dockerClient.getContainer(containerId);
        if (!inspect.isPresent()) {
            return null;
        }

        var hostConfig = inspect.get().getHostConfig();
        if (hostConfig.isHostNetwork()) {
            return LOOPBACK_ADDRESS;
        }

        var containerNetwork = hostConfig.getContainerNetwork();
        if (containerNetwork != null) {
            return this.getContainerGatewayFromDockerInspect(containerNetwork, mustBeBridge);
        }

        var gateway = inspect.get().getNetworkSettings().getGateway();
        if (gateway == null) {
            gateway = inspect.get()
                    .getNetworkSettings()
                    .getNetworks()
                    .values()
                    .stream()
                    .map(ContainerNetwork::getGateway)
                    .filter(Objects::nonNull)
                    .findAny()
                    .orElse(null);
        }

        return (mustBeBridge && !this.isBridge(gateway)) ? null : gateway;
    }

    private boolean isBridge(String gateway) {
        return gateway != null && ContainerNetworkUtil.isBridgeNetworkInterfaceAddress(gateway);
    }

    private List<String> getContainerAdresses() {
        List<String> ips = new ArrayList<>(this.getContainerAddressesFromDockerInspect(this.vm.getContainerId()));
        ips.addAll(this.getContainerAdressessFromProc(this.vm.getPid()));
        return ips;
    }

    private List<String> getContainerAddressesFromDockerInspect(String containerId) {
        var inspect = this.dockerClient.getContainer(containerId);
        if (!inspect.isPresent()) {
            return Collections.emptyList();
        }

        var hostConfig = inspect.get().getHostConfig();
        if (hostConfig.isHostNetwork()) {
            return Collections.singletonList(LOOPBACK_ADDRESS);
        }

        var containerNetwork = hostConfig.getContainerNetwork();
        if (containerNetwork != null) {
            return this.getContainerAddressesFromDockerInspect(containerNetwork);
        }

        List<String> addresses = new ArrayList<>();
        if (inspect.get().getNetworkSettings().getIpAddress() != null) {
            addresses.add(inspect.get().getNetworkSettings().getIpAddress());
        }

        for (var network : inspect.get().getNetworkSettings().getNetworks().values()) {
            if (network.getIpAddress() != null) {
                addresses.add(network.getIpAddress());
            }
        }
        return addresses;
    }

    private Collection<String> getContainerAdressessFromProc(int pid) {
        try {
            return ProcFs.ROOT.readIpAddressesFromForwardingInformationBase(pid);
        } catch (IOException ex) {
            this.log.debug("Could not read ip addresses from proc for {}", pid);
            return Collections.emptyList();
        }
    }

    private static Map<String, String> getEnvOverrides() {
        Map<String, String> map = new HashMap<>();
        map.put("_JAVA_OPTIONS", "");
        map.put("JAVA_TOOL_OPTIONS", "");
        return map;
    }
}
