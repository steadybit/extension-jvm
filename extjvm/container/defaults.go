package container

const (
	runtimeContainerd         runtime = "containerd"
	defaultSocketContainerd           = "/run/containerd/containerd.sock"
	defaultRuncRootContainerd         = "/run/containerd/runc/k8s.io"
	RuntimeDocker             runtime = "docker"
	defaultSocketDocker               = "/var/run/docker.sock"
	defaultRuncRootDocker             = "/run/docker/runtime-runc/moby"
	runtimeCrio               runtime = "cri-o"
	defaultSocketCrio                 = "/var/run/crio/crio.sock"
	defaultRuncRootCrio               = "/run/runc"
)

type runtime string

var (
	allRuntimes = []runtime{RuntimeDocker, runtimeContainerd, runtimeCrio}
)

func (r runtime) defaultSocket() string {
	switch r {
	case RuntimeDocker:
		return defaultSocketDocker
	case runtimeContainerd:
		return defaultSocketContainerd
	case runtimeCrio:
		return defaultSocketCrio
	}
	return ""
}

func (r runtime) defaultRuncRoot() string {
	switch r {
	case RuntimeDocker:
		return defaultRuncRootDocker
	case runtimeContainerd:
		return defaultRuncRootContainerd
	case runtimeCrio:
		return defaultRuncRootCrio
	}
	return ""
}
