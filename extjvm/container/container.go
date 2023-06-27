package container

import (
  "github.com/steadybit/extension-jvm/extjvm/container/types"
  "os"
)

func autoDetectContainerRuntime() (runtime types.Runtime) {
  for _, r := range types.AllRuntimes {
    if _, err := os.Stat(r.DefaultSocket()); err == nil {
      return r
    }
  }
  return ""
}


func GetRuncRoot() string {
  runtime := autoDetectContainerRuntime()
  return runtime.DefaultRuncRoot()
}
