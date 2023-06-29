package hsperf

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"path/filepath"
	"strings"
)

func FindHsPerfDataDirs(dirsGlob string) map[string]string {
	log.Trace().Msgf("Looking for hsperfdata files in %s", dirsGlob)
	filePaths := make(map[string]string)
	cmd := utils.RootCommandContext(context.Background(), "find", dirsGlob, "-maxdepth", "1", "-type", "d", "-name", "hsperfdata_*", "-exec", "find", "{}", "-name", "?", ";")
	output, err := cmd.Output()
	if err != nil {
		log.Error().Msgf("Error while globbing %s: %s", dirsGlob, err)
		return filePaths
	}
	paths := strings.Split(string(output), "\n")
	for _, path := range paths {
		if path == "" {
			continue
		}
		pid := filepath.Base(path)
		log.Trace().Msgf("Found hsperfdata file for pid %s: %s", pid, path)
		filePaths[pid] = path
	}

	return filePaths
}

func IsAttachable(entryMap map[string]interface{}) bool {
	capabilities := GetStringProperty(entryMap, "sun.rt.jvmCapabilities")
	if capabilities == "" {
		return false
	}
	if capabilities[0] == '1' {
		return true
	}
	return false
}

func GetStringProperty(entryMap map[string]interface{}, key string) string {
	if value, ok := entryMap[key]; ok {
		return value.(string)
	}
	if value, ok := entryMap[fmt.Sprintf("java.property.%s", key)]; ok {
		return value.(string)
	}
	log.Error().Msgf("Could not get property %s from perfdata", key)
	return ""
}
