<img src="./logo.svg" height="130" align="right" alt="JVM logo">

# Steadybit extension-jvm

This [Steadybit](https://www.steadybit.com/) extension provides a jvm instance discovery and the various actions for jvm instances targets.

Learn about the capabilities of this extension in our [Reliability Hub](https://hub.steadybit.com/extension/com.steadybit.extension_jvm).

## Configuration

| Environment Variable                                    | Helm value                                             | Meaning                                                                                                                                                            | Required | Default |
|---------------------------------------------------------|--------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------|---------|
| `STEADYBIT_EXTENSION_RUNTIME`                           | `container.runtime`                                    | The container runtime to user either `docker`, `containerd` or `cri-o`. Will be automatically configured if not specified.                                         | yes      | (auto)  |
| `STEADYBIT_EXTENSION_SOCKET`                            | `containerRuntimes.(docker/containerd/cri-o).socket`   | The socket used to connect to the container runtime. Will be automatically configured if not specified.                                                            | yes      | (auto)  |
| `STEADYBIT_EXTENSION_CONTAINERD_NAMESPACE`              |                                                        | The containerd namespace to use.                                                                                                                                   | yes      | k8s.io  |
| `STEADYBIT_EXTENSION_RUNC_ROOT`                         | `containerRuntimes.(docker/containerd/cri-o).runcRoot` | The runc root to use.                                                                                                                                              | yes      | (auto)  |
| `STEADYBIT_EXTENSION_RUNC_DEBUG`                        |                                                        | Activate debug mode for run.                                                                                                                                       |          |         |
| `STEADYBIT_EXTENSION_JVM_ATTACHMENT_ENABLED`            |                                                        | is jvm attachment enabled                                                                                                                                          | no       | true    |
| `STEADYBIT_EXTENSION_JAVA_AGENT_ATTACHMENT_PORT`        |                                                        | java agent attachment port                                                                                                                                         | no       | 8095    |
| `STEADYBIT_EXTENSION_CONTAINER_ADDRESS`                 |                                                        | public ip of the extension                                                                                                                                         | no       |         |
| `STEADYBIT_EXTENSION_DISCOVERY_ATTRIBUTES_EXCLUDES_JVM` | `discovery.attributes.excludes.jvm`                    | List of Target Attributes which will be excluded during discovery. Checked by key equality and supporting trailing "*"                                             | false    |         |

The extension supports all environment variables provided by [steadybit/extension-kit](https://github.com/steadybit/extension-kit#environment-variables).

## Needed capabilities

The capabilities needed by this extension are: (which are provided by the helm chart)

- SYS_ADMIN
- SYS_RESOURCE
- SYS_PTRACE
- KILL
- NET_ADMIN
- DAC_OVERRIDE
- SETUID
- SETGID
- AUDIT_WRITE

## Installation

### Using Helm in Kubernetes

```sh
$ helm repo add steadybit-extension-jvm https://steadybit.github.io/extension-jvm
$ helm repo update
$ helm upgrade steadybit-extension-jvm \
    --install \
    --wait \
    --timeout 5m0s \
    --create-namespace \
    --namespace steadybit-extension \
    --set container.runtime=docker \
    steadybit-extension-jvm/steadybit-extension-jvm
```

### Using Docker

```sh
docker run \
  --rm \
  -p 8087 \
  --privileged \
  --pid=host \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /run/docker/runtime-runc/moby:/run/docker/runtime-runc/moby\
  -v /sys/fs/cgroup:/sys/fs/cgroup\
  --name steadybit-extension-jvm \
  ghcr.io/steadybit/extension-jvm:latest
```

### Linux Package

Please use our [agent-linux.sh script](https://docs.steadybit.com/install-and-configure/install-agent/install-on-linux-hosts) to install the extension on your Linux machine.
The script will download the latest version of the extension and install it using the package manager.

## Register the extension

Make sure to register the extension at the steadybit platform. Please refer to
the [documentation](https://docs.steadybit.com/integrate-with-steadybit/extensions/extension-installation) for more information.

## Anatomy of the extension / Security

We try to limit the needed access needed for the extension to the absolute minimum. So the extension itself can run as a non-root user on a read-only root file-system and will by default if deployed using the provided helm-chart.
In order do execute certain actions the extension needs certain capabilities.

### discovery / state attacks

For discovery the extension needs access to the container runtime socket.

### resource and network attacks

The jvm attachment reuses the target container's linux namespace(s), control group(s) and user.
This requires the following capabilities: SYS_ADMIN, SYS_RESOURCE, SYS_PTRACE, KILL, NET_ADMIN, DAC_OVERRIDE, SETUID, SETGID, AUDIT_WRITE.

#### How do I exclude my JVM from the discovery mechanism?

Add the `steadybit.agent.disable-jvm-attachment` flag to your JVM commandline like in this example:

```
java -Dsteadybit.agent.disable-jvm-attachment -jar spring-boot-sample.jar --server.port=0
```
