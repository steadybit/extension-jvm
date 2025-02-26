package jvm

import (
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/steadybit/extension-jvm/extjvm/jvm/starttime"
	"github.com/steadybit/extension-kit/extruntime"
	"strings"
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
	StartTime() starttime.Time
	VmArgs() string
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
	startTime     starttime.Time
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

func (vm defaultJavaVm) VmArgs() string {
	return vm.vmArgs
}

func (vm defaultJavaVm) IsRunning() bool {
	p2, err := process.NewProcess(vm.p.Pid)
	if errors.Is(err, process.ErrorProcessNotRunning) {
		return false
	}
	startTime2, err := starttime.ForProcess(p2)
	if err != nil {
		return false
	}
	return vm.startTime == startTime2
}

func newJavaVm(p *process.Process, via string) *defaultJavaVm {
	cmdline, _ := p.Cmdline()
	exePath, _ := p.Exe()

	hostname, fqdn, _ := extruntime.GetHostname()

	vm := &defaultJavaVm{
		p:             p,
		commandLine:   cmdline,
		path:          exePath,
		hostname:      hostname,
		hostFQDN:      fqdn,
		discoveredVia: via,
	}

	if startTime, err := starttime.ForProcess(p); err == nil {
		vm.startTime = startTime
	} else {
		log.Debug().Err(err).Msgf("Failed to get starttime for pid %d", p.Pid)
	}

	if uids, err := p.Uids(); err == nil && len(uids) > 0 {
		vm.userId = fmt.Sprintf("%d", uids[0])
	}

	if gids, err := p.Gids(); err == nil && len(gids) > 0 {
		vm.groupId = fmt.Sprintf("%d", gids[0])
	}

	if processPath, err := getProcessPath(context.Background(), p); err == nil && processPath != "" {
		vm.path = processPath
	} else {
		log.Debug().Err(err).Msgf("Failed to get process path for pid %d", p.Pid)
	}

	args := strings.Split(cmdline, " ")
	for i, arg := range args {
		if arg == "-cp" || arg == "-classpath" {
			vm.classPath = args[i+1]
			break
		}
	}

	return vm
}

func (vm defaultJavaVm) StartTime() starttime.Time {
	return vm.startTime
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
