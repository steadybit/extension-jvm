package jvm

import "fmt"

func (vm JavaVm) ToDebugString() string {
	return fmt.Sprintf("JavaVm{pid=%d, discoveredVia=%s, commandLine=%s, mainClass=%s, classpath=%s, containerId=%s, inContainerPid=%d, vmVersion=%s, vmVendor=%s, vmName=%s, vmArgs=%s, userId=%s, groupId=%s, path=%s}",
		vm.Pid, vm.DiscoveredVia, vm.CommandLine, vm.MainClass, vm.ClassPath, vm.ContainerId, vm.InContainerPid, vm.VmVersion, vm.VmVendor, vm.VmName, vm.VmArgs, vm.UserId, vm.GroupId, vm.Path)
}

func (vm JavaVm) ToInfoString() string {
	return fmt.Sprintf("JavaVm{pid=%d, discoveredVia=%s, mainClass=%s,  containerId=%s}",
		vm.Pid, vm.DiscoveredVia, vm.MainClass, vm.ContainerId)
}

func (vm JavaVm) IsRunningInContainer() bool {
	return vm.ContainerId != ""
}

type JavaVm struct {
	Pid            int32
	CommandLine    string
	MainClass      string
	ClassPath      string
	ContainerId    string
	InContainerPid int
	VmVersion      string
	VmVendor       string
	VmName         string
	VmArgs         string
	UserId         string
	GroupId        string
	Path           string
	DiscoveredVia  string
	Hostname       string
	HostFQDN       string
}
