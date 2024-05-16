// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package hsperf

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/v3/process"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"github.com/xin053/hsperfdata"
	"k8s.io/utils/path"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func FindHsperfdataFileContainer(ctx context.Context, p *process.Process, pidInContainer int32) string {
	containerFs := filepath.Join("/proc", strconv.Itoa(int(p.Pid)), "root")

	hsPerfDataPaths := []string{
		filepath.Join(containerFs, os.TempDir()),
	}

	if alternateTempDir := findAlternateTempDir(p); alternateTempDir != "" {
		hsPerfDataPaths = append(hsPerfDataPaths, filepath.Join(containerFs, alternateTempDir))
	}

	return findHsperfdataFile(ctx, pidInContainer, hsPerfDataPaths...)
}

func FindHsperfdataFile(ctx context.Context, p *process.Process) string {
	hsPerfDataPaths := []string{os.TempDir()}

	if runtime.GOOS == "linux" {
		hsPerfDataPaths = append(hsPerfDataPaths, filepath.Join("/proc", strconv.Itoa(int(p.Pid)), "root", os.TempDir()))
	}

	if alternateTempDir := findAlternateTempDir(p); alternateTempDir != "" {
		hsPerfDataPaths = append(hsPerfDataPaths, alternateTempDir)
	}

	return findHsperfdataFile(ctx, p.Pid, hsPerfDataPaths...)
}

func findHsperfdataFile(ctx context.Context, pid int32, paths ...string) string {
	var allErrs []error
	for _, p := range paths {
		if f, err := findHsPerfDataFile(ctx, p, pid); err != nil {
			allErrs = append(allErrs, err)
		} else {
			return f
		}
	}
	if len(allErrs) > 0 {
		log.Trace().Errs("err", allErrs).Int32("pid", pid).Strs("paths", paths).Msgf("failed to find hsperfdata fila")
	}
	return ""
}

func findHsPerfDataFile(ctx context.Context, searchDir string, pid int32) (string, error) {
	glob := filepath.Join(searchDir, "hsperfdata_*")
	log.Trace().Msgf("Search for hsperfdata  for %d in %s", pid, glob)

	output, err := utils.RootCommandContext(ctx, "sh", "-c", fmt.Sprintf("find %s -type f -name %d -maxdepth 1", glob, pid)).Output()
	if err != nil {
		return "", fmt.Errorf("hsperfdata file not found in %s: %w", glob, err)
	}

	match := ""
	for _, p := range strings.Split(string(output), "\n") {
		if p == "" {
			continue
		}
		if match == "" {
			log.Trace().Str("match", match).Int32("pid", pid).Msgf("found hsperfdata file for pid %d: %s", pid, p)
			match = p
		} else {
			log.Warn().Int32("pid", pid).Str("match", match).Str("match2", p).Msgf("found ambigupus hsperfdata files")
		}
	}

	return match, nil
}

type Data struct {
	entries map[string]interface{}
}

func ReadData(file string) (Data, error) {
	entries, err := hsperfdata.ReadPerfData(file, false)
	if err == nil {
		return Data{entries: entries}, nil
	} else {
		return Data{entries: make(map[string]interface{})}, err
	}
}

func (d Data) IsAttachable() bool {
	capabilities := d.GetStringProperty("sun.rt.jvmCapabilities")
	if capabilities == "" {
		return false
	}
	if capabilities[0] == '1' {
		return true
	}
	return false
}

func (d Data) GetStringProperty(key string) string {
	if value, ok := d.entries[key]; ok {
		return value.(string)
	}
	if value, ok := d.entries[fmt.Sprintf("java.property.%s", key)]; ok {
		return value.(string)
	}
	log.Error().Msgf("Could not get property %s from perfdata", key)
	return ""
}

func findAlternateTempDir(p *process.Process) string {
	if cmdline, err := p.CmdlineSlice(); err == nil {
		for _, arg := range cmdline {
			if !strings.HasPrefix(arg, "-Djava.io.tmpdir") {
				continue
			}
			tokens := strings.SplitN(arg, "=", 2)
			if len(tokens) <= 1 {
				continue
			}
			if ok, _ := path.Exists(path.CheckSymlinkOnly, tokens[1]); ok {
				return tokens[1]
			}
		}
	}
	return ""
}
