package common

import (
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
)

func GetJavaagentPath() string {
	pathByEnv := os.Getenv("STEADYBIT_EXTENSION_JAVA_AGENT_PATH")
	if pathByEnv != "" {
		return pathByEnv
	}
	return "javaagents/target/javaagent"
}

func GetJarPath(jarName string) string {
	p := GetJavaagentPath()
	abs, err := filepath.Abs(filepath.Join(p, jarName))
	if err != nil {
		log.Err(err).Msgf("Failed to get absolute path for %s", jarName)
		return ""
	}
	return abs
}
