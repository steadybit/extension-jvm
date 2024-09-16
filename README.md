<img src="./logo.svg" height="130" align="right" alt="JVM logo">

# Steadybit extension-jvm

This [Steadybit](https://www.steadybit.com/) extension provides a jvm instance discovery and the various actions for jvm
instances targets.

Learn about the capabilities of this extension in
our [Reliability Hub](https://hub.steadybit.com/extension/com.steadybit.extension_jvm).

## Configuration

| Environment Variable                                    | Helm value                                             | Meaning                                                                                                                    | Required | Default |
|---------------------------------------------------------|--------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------|----------|---------|
| `STEADYBIT_EXTENSION_RUNTIME`                           | `container.runtime`                                    | The container runtime to user either `docker`, `containerd` or `cri-o`. Will be automatically configured if not specified. | yes      | (auto)  |
| `STEADYBIT_EXTENSION_SOCKET`                            | `containerRuntimes.(docker/containerd/cri-o).socket`   | The socket used to connect to the container runtime. Will be automatically configured if not specified.                    | yes      | (auto)  |
| `STEADYBIT_EXTENSION_CONTAINERD_NAMESPACE`              |                                                        | The containerd namespace to use.                                                                                           | yes      | k8s.io  |
| `STEADYBIT_EXTENSION_RUNC_ROOT`                         | `containerRuntimes.(docker/containerd/cri-o).runcRoot` | The runc root to use.                                                                                                      | yes      | (auto)  |
| `STEADYBIT_EXTENSION_RUNC_DEBUG`                        |                                                        | Activate debug mode for run.                                                                                               |          |         |
| `STEADYBIT_EXTENSION_JVM_ATTACHMENT_ENABLED`            |                                                        | is jvm attachment enabled                                                                                                  | no       | true    |
| `STEADYBIT_EXTENSION_JAVA_AGENT_ATTACHMENT_PORT`        |                                                        | java agent attachment port                                                                                                 | no       | 8095    |
| `STEADYBIT_EXTENSION_CONTAINER_ADDRESS`                 |                                                        | public ip of the extension                                                                                                 | no       |         |
| `STEADYBIT_EXTENSION_DISCOVERY_ATTRIBUTES_EXCLUDES_JVM` | `discovery.attributes.excludes.jvm`                    | List of Target Attributes which will be excluded during discovery. Checked by key equality and supporting trailing "*"     | false    |         |

The extension supports all environment variables provided
by [steadybit/extension-kit](https://github.com/steadybit/extension-kit#environment-variables).

When installed as linux package this configuration is in`/etc/steadybit/extension-jvm`.

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

## Needed access to the java process

To be able to discover we need access to the java process. This is done by using the `attach` mechanism of the JVM. This
is a standard mechanism and is used by many other tools.
You may see a warning in the logs of the JVM that the extension is attaching to the JVM. This is normal and expected.
It will look like this:

```
WARNING: A Java agent has been loaded dynamically (...javaagent-init.jar)
WARNING: Dynamic loading of agents will be disallowed by default in a future release
```

To avoid this warning or be able to use this extension in future java releases you can use the
`-XX:+EnableDynamicAgentLoading` flag in your JVM commandline to be able to load the javaagent dynamically.

## Installation

### Kubernetes

Detailed information about agent and extension installation in kubernetes can also be found in
our [documentation](https://docs.steadybit.com/install-and-configure/install-agent/install-on-kubernetes).

#### Recommended (via agent helm chart)

All extensions provide a helm chart that is also integrated in the
[helm-chart](https://github.com/steadybit/helm-charts/tree/main/charts/steadybit-agent) of the agent.

The extension is installed by default when you install the agent.

You can provide additional values to configure this extension.

```
--set extension-jvm.container.runtime={{YOUR-CONTAINER-RUNTIME}} \
```

Additional configuration options can be found in
the [helm-chart](https://github.com/steadybit/extension-jvm/blob/main/charts/steadybit-extension-jvm/values.yaml) of the
extension.

#### Alternative (via own helm chart)

If you need more control, you can install the extension via its
dedicated [helm-chart](https://github.com/steadybit/extension-jvm/blob/main/charts/steadybit-extension-jvm).

```bash
helm repo add steadybit-extension-jvm https://steadybit.github.io/extension-jvm
helm repo update
helm upgrade steadybit-extension-jvm \
    --install \
    --wait \
    --timeout 5m0s \
    --create-namespace \
    --namespace steadybit-agent \
    --set container.runtime=docker \
    steadybit-extension-jvm/steadybit-extension-jvm
```

### Linux Package

Please use
our [agent-linux.sh script](https://docs.steadybit.com/install-and-configure/install-agent/install-on-linux-hosts)
to install the extension on your Linux machine. The script will download the latest version of the extension and install
it using the package manager.

After installing, configure the extension by editing `/etc/steadybit/extension-jvm` and then restart the service.

## Extension registration

Make sure that the extension is registered with the agent. In most cases this is done automatically. Please refer to
the [documentation](https://docs.steadybit.com/install-and-configure/install-agent/extension-discovery) for more
information about extension registration and how to verify.

## Anatomy of the extension / Security

We try to limit the needed access needed for the extension to the absolute minimum. So the extension itself can run as a
non-root user on a read-only root file-system and will by default if deployed using the provided helm-chart.
In order do execute certain actions the extension needs certain capabilities.

### discovery / state attacks

For discovery the extension needs access to the container runtime socket.

### resource and network attacks

The jvm attachment reuses the target container's linux namespace(s), control group(s) and user.
This requires the following capabilities: SYS_ADMIN, SYS_RESOURCE, SYS_PTRACE, KILL, NET_ADMIN, DAC_OVERRIDE, SETUID,
SETGID, AUDIT_WRITE.

## FAQ

#### How do I exclude my JVM from the discovery mechanism?

Add the `steadybit.agent.disable-jvm-attachment` flag to your JVM commandline like in this example:

```
java -Dsteadybit.agent.disable-jvm-attachment -jar spring-boot-sample.jar --server.port=0
```

#### The Spring discovery is not showing my application name

The extension uses the `spring.application.name` property to determine the application name. The extension will try to
lookup the property by the JMX Environment.

If JMX is not enabled or the values are sanitized
by [Spring Boot](https://docs.spring.io/spring-boot/reference/actuator/endpoints.html#actuator.endpoints.sanitization),
the extension will try to instrument the application context and get the value from there. This might not work in an
unused application, as the instrumentation needs some events to discover the application context.

Workarounds:
 - Calling any HTTP endpoint is sufficient to trigger the application context instrumentation.
 - Provide a [SanitizingFunction](https://docs.spring.io/spring-boot/how-to/actuator.html#howto.actuator.customizing-sanitization) and exclude the `spring.application.name` from sanitization.
 - Set `management.endpoint.env.show-values` to `always` to expose the environment values. (Be careful, this will disable springs default sanitization!)
