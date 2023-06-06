package hsperf

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"path/filepath"
)

func FindHsPerfDataDirs(dirsGlob string) map[string]string {
	filePaths := make(map[string]string)
	paths, err := filepath.Glob(dirsGlob)
	if err != nil {
		log.Error().Msgf("Error while globbing %s: %s", dirsGlob, err)
		return filePaths
	}

	for _, path := range paths {
		pid := filepath.Base(path)
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
