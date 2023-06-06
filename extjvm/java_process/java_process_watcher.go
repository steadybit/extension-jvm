package java_process

import (
  "context"
  "github.com/procyon-projects/chrono"
  "github.com/rs/zerolog/log"
  "github.com/shirou/gopsutil/process"
  "github.com/steadybit/extension-jvm/extjvm/utils"
  "strings"
  "time"
)

type Listener interface {
  NewProcess(p *process.Process)
}

var pids []int32
var listeners []Listener

var RunningStates = []string{"R", "W", "S"} // Running, Waiting, Sleeping

func Start() {
  taskScheduler := chrono.NewDefaultTaskScheduler()

  _, err := taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
    updatePids()
  }, 5 * time.Second)

  if err == nil {
    log.Info().Msg("Java Process Watcher Task has been scheduled successfully.")
  }
}
func updatePids() {
  processes, err := process.Processes()
  if err != nil {
    return
  }
  if err != nil {
    log.Error().Err(err).Msg("Failed to list processes")
    return
  }
  for _, p := range processes {
    if !utils.Contains(pids, p.Pid) {
      if IsRunning(p) {
        pids = append(pids, p.Pid)
        newProcess(p)
      }
    }
  }
}

func newProcess(p *process.Process) {
  if isJava(p) {
    for _, listener := range listeners {
      listener.NewProcess(p)
    }
  }
}

func AddListener(listener Listener) {
  listeners = append(listeners, listener)
  for _, pid := range pids {
    p, err := process.NewProcess(pid)
    if err != nil {
      log.Warn().Err(err).Msg("Error in listener for newProcess")
      continue
    }
    if isJava(p) {
      listener.NewProcess(p)
    }
  }
}

func RemoveListener(listener Listener) {
  for i, l := range listeners {
    if l == listener {
      listeners = append(listeners[:i], listeners[i+1:]...)
      break
    }
  }
}

func isJava(p *process.Process) bool {
  exe, err := p.Exe()
  if err != nil {
    log.Warn().Err(err).Msg("Failed to get process exe")
    return false
  }
  return strings.HasSuffix(strings.TrimSpace(exe), "java")
}

func IsRunning(p *process.Process) bool {
  status, err := p.Status()
  if err != nil {
    log.Trace().Err(err).Msg("Failed to get process status")
    return false
  }
  return utils.ContainsString(RunningStates, status)
}
