package java_process

import (
  "context"
  "github.com/procyon-projects/chrono"
  "github.com/rs/zerolog/log"
  "github.com/shirou/gopsutil/process"
  "github.com/steadybit/extension-jvm/extjvm/utils"
  "path/filepath"
  "strconv"
  "strings"
  "time"
)

type Listener interface {
	NewProcess(p *process.Process)
}

var (
	pids          []int32
	listeners     []Listener
	RunningStates = []string{"R", "W", "S"} // Running, Waiting, Sleeping
)

func Start() {
	taskScheduler := chrono.NewDefaultTaskScheduler()

	_, err := taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		updatePids()
	}, 5*time.Second)

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
  path, err := GetProcessPath(p)
  if err != nil {
    return false
  }
  return strings.HasSuffix(strings.TrimSpace(path), "java")
}

func GetProcessPath(p *process.Process) (string, error) {
  exePath := filepath.Join("/proc", strconv.Itoa(int(p.Pid)), "exe")
  cmd := utils.RootCommandContext(context.Background(), "readlink", exePath)
  output, err := cmd.Output()
  if err != nil {
    exe, err := p.Exe()
    if err != nil {
      log.Error().Err(err).Msg("Failed to get process exe")
      return "", err
    }
    output = []byte(exe)
  }
  return strings.Trim(string(output), "\n") , nil
}

func IsRunning(p *process.Process) bool {
	status, err := p.Status()
	if err != nil {
		log.Trace().Err(err).Msg("Failed to get process status")
		return false
	}
	return utils.ContainsString(RunningStates, status)
}

func IsRunningProcess(pid int32) bool {
  p, err := process.NewProcess(pid)
  if err != nil {
    log.Trace().Err(err).Msg("Process not running anymore")
    return false
  }
  return IsRunning(p)
}
