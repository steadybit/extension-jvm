package jvm

import (
	"fmt"
	"github.com/shirou/gopsutil/v3/process"
	"time"
)

type JavaVm interface {
	Pid() int32
	CommandLine() string
	MainClass() string
	ClassPath() string
	UserId() string
	GroupId() string
	Path() string
	Hostname() string
	HostFQDN() string
	IsRunning() bool
	ToInfoString() string
	ToDebugString() string
	CreateTime() time.Time
}

type JavaVmInContainer interface {
	JavaVm
	ContainerId() string
	PidInContainer() int32
}

type defaultJavaVm struct {
	p             *process.Process
	commandLine   string
	mainClass     string
	classPath     string
	vmVersion     string
	vmVendor      string
	vmName        string
	vmArgs        string
	userId        string
	groupId       string
	path          string
	discoveredVia string
	hostname      string
	hostFQDN      string
}

func (vm defaultJavaVm) Pid() int32 {
	return vm.p.Pid
}

func (vm defaultJavaVm) CommandLine() string {
	return vm.commandLine
}

func (vm defaultJavaVm) MainClass() string {
	return vm.mainClass
}

func (vm defaultJavaVm) ClassPath() string {
	return vm.classPath
}

func (vm defaultJavaVm) UserId() string {
	return vm.userId
}

func (vm defaultJavaVm) GroupId() string {
	return vm.groupId
}

func (vm defaultJavaVm) Path() string {
	return vm.path
}

func (vm defaultJavaVm) Hostname() string {
	return vm.hostname
}

func (vm defaultJavaVm) HostFQDN() string {
	return vm.hostFQDN
}

func (vm defaultJavaVm) IsRunning() bool {
	running, _ := vm.p.IsRunning()
	return running
}

func (vm defaultJavaVm) CreateTime() time.Time {
	createTime, _ := vm.p.CreateTime()
	return time.UnixMilli(createTime)
}

func (vm defaultJavaVm) ToDebugString() string {
	return fmt.Sprintf("JavaVm{pid=%d, discoveredVia=%s, commandLine=%s, mainClass=%s, classpath=%s, vmVersion=%s, vmVendor=%s, vmName=%s, vmArgs=%s, userId=%s, groupId=%s, path=%s}",
		vm.p.Pid, vm.discoveredVia, vm.commandLine, vm.mainClass, vm.classPath, vm.vmVersion, vm.vmVendor, vm.vmName, vm.vmArgs, vm.userId, vm.groupId, vm.path)
}

func (vm defaultJavaVm) ToInfoString() string {
	return fmt.Sprintf("JavaVm{pid=%d, discoveredVia=%s, mainClass=%s}",
		vm.p.Pid, vm.discoveredVia, vm.mainClass)
}

type defaultJavaVmInContainer struct {
	defaultJavaVm
	containerId    string
	pidInContainer int32
}

func (vm defaultJavaVmInContainer) ContainerId() string {
	return vm.containerId
}

func (vm defaultJavaVmInContainer) PidInContainer() int32 {
	return vm.pidInContainer
}

func (vm defaultJavaVmInContainer) ToDebugString() string {
	return fmt.Sprintf("JavaVm{pid=%d, containerId=%s, pidInContainer=%d, discoveredVia=%s, commandLine=%s, mainClass=%s, classpath=%s, vmVersion=%s, vmVendor=%s, vmName=%s, vmArgs=%s, userId=%s, groupId=%s, path=%s}",
		vm.p.Pid, vm.containerId, vm.pidInContainer, vm.discoveredVia, vm.commandLine, vm.mainClass, vm.classPath, vm.vmVersion, vm.vmVendor, vm.vmName, vm.vmArgs, vm.userId, vm.groupId, vm.path)
}

func (vm defaultJavaVmInContainer) ToInfoString() string {
	return fmt.Sprintf("JavaVm{pid=%d, containerId=%s, discoveredVia=%s, mainClass=%s}",
		vm.p.Pid, vm.containerId, vm.discoveredVia, vm.mainClass)
}
