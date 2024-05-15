package utils

import (
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
)

func getJavaagentPath() string {
	pathByEnv := os.Getenv("STEADYBIT_EXTENSION_JAVA_AGENT_PATH")
	if pathByEnv != "" {
		return pathByEnv
	}
	return "javaagents/download/target/javaagent"
}

func GetJarPath(jarName string) string {
	p := getJavaagentPath()
	abs, err := filepath.Abs(filepath.Join(p, jarName))
	if err != nil {
		log.Err(err).Msgf("Failed to get absolute path for %s", jarName)
		return ""
	}
	return abs
}
