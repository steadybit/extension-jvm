package hotspot

import (
  "github.com/steadybit/extension-jvm/extjvm/hsperf"
  "github.com/steadybit/extension-kit/extutil"
  "os"
  "path/filepath"
  "strconv"
)

func GetJvmPids(hostPid int32, rootPath string) []int32 {
  filePaths := GetRootHsPerfPaths(hostPid, rootPath)

  jvmPids := make([]int32, len(filePaths))
  for pid, _ := range filePaths {
    jvmPids = append(jvmPids, extutil.ToInt32(pid))
  }
  return jvmPids
}

func GetRootHsPerfPaths(hostPid int32, rootPath string) map[string]string {
  dirsGlob := filepath.Join(rootPath, os.TempDir(), "hsperfdata_*", strconv.Itoa(int(hostPid)))
  filePaths := hsperf.FindHsPerfDataDirs(dirsGlob)
  return filePaths
}
