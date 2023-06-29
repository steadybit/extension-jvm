package hotspot

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/hsperf"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/xin053/hsperfdata"
	"os"
	"path/filepath"
)

func GetJvmPids() []int32 {
	filePaths, err := hsperfdata.AllPerfDataPaths()
	if err != nil {
		log.Err(err).Msg("Failed to list hsperfdata paths")
		return nil
	}

	jvmPids := make([]int32, 0)
	for pid := range filePaths {
		jvmPids = append(jvmPids, extutil.ToInt32(pid))
	}
	return jvmPids
}

func GetJvmPidsForPath(hostPid int32, rootPath string) []int32 {
	filePaths := GetRootHsPerfPaths(hostPid, rootPath)

	jvmPids := make([]int32, len(filePaths))
	for pid := range filePaths {
		jvmPids = append(jvmPids, extutil.ToInt32(pid))
	}
	return jvmPids
}

func GetRootHsPerfPaths(hostPid int32, rootPath string) map[string]string {
	log.Trace().Msgf("Looking for hsperfdata files in %s with hostPid %d", rootPath, hostPid)
	filePaths := hsperf.FindHsPerfDataDirs(filepath.Join(rootPath, os.TempDir()))
	return filePaths
}
