package utils

import (
  "context"
  "os/exec"
  "strings"
  "sync"
  "syscall"
  "time"
)

func Contains(s []int32, str int32) bool {
  for _, v := range s {
    if v == str {
      return true
    }
  }
  return false
}

func ContainsString(s []string, str string) bool {
  for _, v := range s {
    if v == str {
      return true
    }
  }
  return false
}

func ContainsPartOfString(s []string, str string) bool {
  for _, v := range s {
    if strings.Contains(str, v) {
      return true
    }
  }
  return false
}

// WaitTimeout waits for the waitgroup for the specified max timeout.
// Returns true if waiting timed out.
func WaitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
  c := make(chan struct{})
  go func() {
    defer close(c)
    wg.Wait()
  }()
  select {
  case <-c:
    return false // completed normally
  case <-time.After(timeout):
    return true // timed out
  }
}

func RootCommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
  cmd := exec.CommandContext(ctx, name, arg...)
  cmd.SysProcAttr = &syscall.SysProcAttr{
    Credential: &syscall.Credential{
      Uid: 0,
      Gid: 0,
    },
  }
  return cmd
}
