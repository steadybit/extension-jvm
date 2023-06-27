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
	"sync"
	"time"
)

type Listener interface {
	NewJavaProcess(p *process.Process) bool
}

type DiscoveryWork struct {
	p       *process.Process
	retries int
}

var (
	discoveredPidsMutex sync.Mutex
	discoveredPids      []int32
  ignoredPidsMutex sync.Mutex
	ignoredPids      []int32
	listeners      []Listener
	RunningStates = []string{"R", "W", "S"} // Running, Waiting, Sleeping

	discoveryJobs = make(chan DiscoveryWork)
)

const (
	initialRetries = 3
)

func Start() {
	taskScheduler := chrono.NewDefaultTaskScheduler()

	_, err := taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		updatePids()
	}, 5*time.Second)

  _, err = taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
    // Clear ignored pids because they might be reused
		ignoredPidsMutex.Lock()
    ignoredPids = []int32{}
    ignoredPidsMutex.Unlock()
	}, 1*time.Hour)

	if err == nil {
		log.Info().Msg("Java Process Watcher Task has been scheduled successfully.")
	}

	// create hotspot discovery worker pool
	for w := 1; w <= 4; w++ {
		go discoveryWorker(discoveryJobs)
	}
}

func discoveryWorker(discoveryJobs chan DiscoveryWork) {
	for job := range discoveryJobs {
		job.retries--
		if job.retries > 0 {
			discoverProcessJVM(job)
		} else {
			log.Trace().Msgf("Process discovery retries for %d exceeded.", job.p.Pid)
      addPidToIgnoredPids(job)
		}
	}
}

func discover(p *process.Process, retries int) {
	if retries != initialRetries {
		time.Sleep(1 * time.Second)
	}
	discoveryJobs <- DiscoveryWork{p: p, retries: retries}
}

func updatePids() {
	processes, err := process.Processes()
	if err != nil {
		log.Error().Err(err).Msg("Failed to list processes")
		return
	}
	for _, p := range processes {
    if utils.Contains(ignoredPids, p.Pid) {
      continue
    }
		discover(p, initialRetries)
	}
}

func discoverProcessJVM(job DiscoveryWork) {
	if !utils.Contains(discoveredPids, job.p.Pid) {
		if IsRunning(job.p) {
      success := notifyListenersForNewProcess(job.p)
      if success {
        addPidToDiscoveredPids(job)
      } else {
        //retry
        discover(job.p, job.retries)
      }
    }
  }
}

func addPidToDiscoveredPids(job DiscoveryWork) {
  discoveredPidsMutex.Lock()
  discoveredPids = append(discoveredPids, job.p.Pid)
  discoveredPidsMutex.Unlock()
}

func addPidToIgnoredPids(job DiscoveryWork) {
  ignoredPidsMutex.Lock()
  ignoredPids = append(ignoredPids, job.p.Pid)
  ignoredPidsMutex.Unlock()
}

func notifyListenersForNewProcess(p *process.Process) bool {
  success := true
	if isJava(p) {
		for _, listener := range listeners {
			currentResult := listener.NewJavaProcess(p)
      if !currentResult {
        success = false
      }
		}
	}
  return success
}

func AddListener(listener Listener) {
	listeners = append(listeners, listener)
	for _, pid := range discoveredPids {
		p, err := process.NewProcess(pid)
		if err != nil {
			log.Warn().Err(err).Msg("Error in listener for notifyListenersForNewProcess")
			continue
		}
		if isJava(p) {
			listener.NewJavaProcess(p)
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
	return strings.Trim(string(output), "\n"), nil
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
