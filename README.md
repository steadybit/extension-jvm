# Steadybit extension-jvm

TODO describe what your extension is doing here from a user perspective.

TODO optionally add your extension to the [Reliability Hub](https://hub.steadybit.com/) by creating
a [pull request](https://github.com/steadybit/reliability-hub-db) and add a link to this README.

## Configuration

| Environment Variable                             | Meaning                    | Required | Default |
|--------------------------------------------------|----------------------------|----------|---------|
| `STEADYBIT_EXTENSION_JVM_ATTACHMENT_ENABLED`     | is jvm attachment enabled  | no       | true    |
| `STEADYBIT_EXTENSION_JAVA_AGENT_ATTACHMENT_PORT` | java agent attachment port | no       | 8095    |
| `STEADYBIT_EXTENSION_CONTAINER_ADDRESS`          | public ip of the extension | no       |         |

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

### Using Docker

```sh
$ docker run \
  --rm \
  -p 8080 \
  --name steadybit-extension-jvm \
  ghcr.io/steadybit/extension-jvm:latest
```

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
    steadybit-extension-jvm/steadybit-extension-jvm
```

## Register the extension

Make sure to register the extension at the steadybit platform. Please refer to
the [documentation](https://docs.steadybit.com/integrate-with-steadybit/extensions/extension-installation) for more information.