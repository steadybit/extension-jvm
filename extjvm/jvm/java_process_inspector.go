// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package jvm

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"codnect.io/chrono"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/steadybit/extension-jvm/chrono_utils"
	"github.com/steadybit/extension-jvm/extjvm/container"
	"github.com/steadybit/extension-jvm/extjvm/jvm/hsperf"
	"github.com/steadybit/extension-jvm/extjvm/utils"
)

type JavaProcessInspector struct {
	JavaVms                    <-chan JavaVm
	ch                         chan<- JavaVm
	scheduler                  chrono.TaskScheduler
	ignoreHsperfData           bool
	minProcessAgeBeforeInspect time.Duration
}

func (i *JavaProcessInspector) Start() {
	i.scheduler = chrono_utils.NewContextTaskScheduler()
	ch := make(chan JavaVm)
	i.ch = ch
	i.JavaVms = ch
}

func (i *JavaProcessInspector) Stop() {
	<-i.scheduler.Shutdown()
	close(i.ch)
}

func (i *JavaProcessInspector) Inspect(p *process.Process, retries int, source string) {
	startTime := time.Now()
	createTime, _ := p.CreateTime()
	age := startTime.Sub(time.UnixMilli(createTime))
	if age < i.minProcessAgeBeforeInspect {
		startTime = startTime.Add(i.minProcessAgeBeforeInspect - age)
	}

	if _, err := i.scheduler.Schedule(func(ctx context.Context) {
		log.Trace().Msgf("Inspecting process %d reported by %s (retries: %d)", p.Pid, source, retries)
		if javaVm, err := i.createJvm(ctx, p, source, retries == 0); err == nil {
			log.Trace().Str("jvm", fmt.Sprintf("%v", javaVm)).Msgf("Successfully inpsected JVM for process %d", p.Pid)
			select {
			case i.ch <- javaVm:
			case <-ctx.Done():
			}
		} else if retries > 0 {
			log.Trace().Err(err).Msgf("Failed to create JVM for process %d. Retrying. Retries left: %d", p.Pid, retries)
			i.Inspect(p, retries-1, source)
		} else {
			log.Warn().Err(err).Msgf("Failed to create JVM for process %d. No more retries left", p.Pid)
		}
	}, chrono.WithTime(startTime)); err != nil {
		log.Warn().Err(err).Msgf("Failed to schedule process insecoption. Pid: %d", p.Pid)
	}
}

func (i *JavaProcessInspector) createJvm(ctx context.Context, p *process.Process, source string, fallbackToProcessOnInaccessibleHsperfdata bool) (JavaVm, error) {
	containerId := container.GetContainerIdForProcess(p)
	if containerId == "" {
		return i.createJvmOnHost(ctx, p, source, fallbackToProcessOnInaccessibleHsperfdata)
	}

	if containerPid := container.FindMappedPidInContainer(p.Pid); containerPid > 0 {
		log.Trace().Msgf("Found JVM %d is running in container %s with in-container-PID %d", p.Pid, containerId, containerPid)
		return i.createJvmInContainer(ctx, p, source, containerId, containerPid, fallbackToProcessOnInaccessibleHsperfdata)
	}

	return nil, errors.New("container pid is not available")
}

func (i *JavaProcessInspector) createJvmOnHost(ctx context.Context, p *process.Process, source string, fallbackToProcessOnInaccessibleHsperfdata bool) (JavaVm, error) {
	var javaVm *defaultJavaVm
	var err error

	if hsPerfDataPath := hsperf.FindHsperfdataFile(ctx, p); hsPerfDataPath != "" {
		log.Trace().Msgf("hsperfdata found for pid %d", p.Pid)
		javaVm, err = i.createJvmUsingHsperfdata(ctx, p, source, hsPerfDataPath)
		if err != nil && !(fallbackToProcessOnInaccessibleHsperfdata && strings.Contains("not accessible", err.Error())) {
			return nil, err
		}
	} else {
		log.Trace().Msgf("No hsperfdata found for pid %d", p.Pid)
	}

	if javaVm == nil {
		javaVm = createJvmFromProcess(p, source)
	}

	return javaVm, err
}

func (i *JavaProcessInspector) createJvmInContainer(ctx context.Context, p *process.Process, source, containerId string, pidInContainer int32, fallbackToProcessOnInaccessibleHsperfdata bool) (JavaVmInContainer, error) {
	var javaVm *defaultJavaVm
	var err error

	if hsperfdataFile := hsperf.FindHsperfdataFileContainer(ctx, p, pidInContainer); hsperfdataFile != "" {
		log.Trace().Msgf("hsperfdata found for pid %d in container %s", pidInContainer, containerId)
		javaVm, err = i.createJvmUsingHsperfdata(ctx, p, source, hsperfdataFile)
		if err != nil && !(fallbackToProcessOnInaccessibleHsperfdata && strings.Contains("not accessible", err.Error())) {
			return nil, err
		}
	} else {
		log.Debug().Msgf("No hsperfdata found for pid %d in container %s", pidInContainer, containerId)
	}

	if javaVm == nil {
		javaVm = createJvmFromProcess(p, source)
	}

	if javaVm == nil {
		return nil, err
	}

	return &defaultJavaVmInContainer{
		defaultJavaVm:  *javaVm,
		containerId:    containerId,
		pidInContainer: pidInContainer,
	}, nil
}

var ErrorNotAttachable = errors.New("not attachable")

func (i *JavaProcessInspector) createJvmUsingHsperfdata(ctx context.Context, p *process.Process, source, path string) (*defaultJavaVm, error) {
	if i.ignoreHsperfData {
		return nil, nil
	}

	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("hsperfdata_%d", p.Pid))
	if err := utils.RootCommandContext(ctx, "cp", path, tempFile).Run(); err != nil {
		return nil, fmt.Errorf("error while copying hsperfdata: %w", err)
	}
	defer func() {
		if err := utils.RootCommandContext(ctx, "rm", tempFile).Run(); err != nil {
			log.Warn().Msgf("Error while removing temp file %s: %s", tempFile, err)
		}
	}()

	data, err := hsperf.ReadData(tempFile)
	if err != nil {
		return nil, fmt.Errorf("error while reading hsperfdata: %w", err)
	}

	if !data.IsAttachable() {
		return nil, ErrorNotAttachable
	}

	vm := createJvmFromProcess(p, source)
	if source == "hsperfdata" {
		vm.discoveredVia = "hsperfdata"
	} else {
		vm.discoveredVia = fmt.Sprintf("%s/hsperfdata", source)
	}

	commandLine := data.GetStringProperty("sun.rt.javaCommand")
	vm.commandLine = commandLine
	vm.mainClass = getMainClass(commandLine)
	vm.classPath = data.GetStringProperty("java.property.java.class.path")
	vm.vmArgs = data.GetStringProperty("java.rt.vmArgs")
	vm.vmName = data.GetStringProperty("java.vm.name")
	vm.vmVendor = data.GetStringProperty("java.vm.vendor")
	vm.vmVersion = data.GetStringProperty("java.vm.version")
	return vm, nil
}

func createJvmFromProcess(p *process.Process, source string) *defaultJavaVm {
	discoveredVia := "os-process"
	if source != "os-process" {
		discoveredVia = fmt.Sprintf("%s/os-process", source)
	}
	return newJavaVm(p, discoveredVia)
}

func getMainClass(commandLine string) string {
	if commandLine == "" {
		return ""
	}
	cmdLine := strings.TrimSpace(commandLine)
	firstSpace := strings.Index(commandLine, " ")
	if firstSpace > 0 {
		cmdLine = cmdLine[:firstSpace]
	}
	/*
	 * Can't use File.separator() here because the separator for the target
	 * jvm may be different from the separator for the monitoring jvm.
	 * And we also strip embedded module e.g. "module/MainClass"
	 */
	lastSlash := strings.LastIndex(cmdLine, "/")
	lastBackslash := strings.LastIndex(cmdLine, "\\")
	lastSeparator := math.Max(float64(lastSlash), float64(lastBackslash))
	if lastSeparator > 0 {
		cmdLine = cmdLine[int(lastSeparator)+1:]
	}
	lastPackageSeparator := strings.LastIndex(cmdLine, ".")
	if lastPackageSeparator > 0 {
		lastPart := cmdLine[int(lastPackageSeparator)+1:]
		/*
		 * We could have a relative path "my.module" or
		 * a module called "my.module" and a jar file called "my.jar" or
		 * class named "jar" in package "my", e.g. "my.jar".
		 * We can never be sure here, but we assume *.jar is a jar file
		 */
		if lastPart == "jar" {
			return cmdLine
		}
		return lastPart
	}
	return cmdLine
}

func getProcessPath(ctx context.Context, p *process.Process) (string, error) {
	exePath := filepath.Join("/proc", strconv.Itoa(int(p.Pid)), "exe")
	if output, err := utils.RootCommandContext(ctx, "readlink", exePath).Output(); err == nil {
		return strings.Trim(string(output), "\n"), nil
	}
	if exe, err := p.Exe(); err == nil {
		return exe, nil
	} else {
		return "", err
	}
}
