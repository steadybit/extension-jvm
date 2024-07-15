package java_process

import (
	"codnect.io/chrono"
	"context"
	"fmt"
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
	discoveredPids   sync.Map
	ignoredPidsMutex sync.Mutex
	ignoredPids      []string
	listeners        []Listener
	RunningStates    = []string{"R", "W", "S", "I", "L"} // Running, Waiting, Sleeping

	discoveryJobs = make(chan DiscoveryWork)
)

const (
	initialRetries = 5
)

func Start() {
	taskScheduler := chrono.NewDefaultTaskScheduler()

	// create discovery worker pool
	for w := 1; w <= 4; w++ {
		go discoveryWorker(discoveryJobs)
	}

	_, err := taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		updatePids()
	}, 5*time.Second)

	if err == nil {
		log.Info().Msg("Java Process Watcher Task has been scheduled successfully.")
	}

	_, err = taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		// Clear ignored pids because they might be reused
		ignoredPidsMutex.Lock()
		ignoredPids = []string{}
		ignoredPidsMutex.Unlock()
	}, 1*time.Hour)

	if err == nil {
		log.Trace().Msg("Cleanup of used pids has been scheduled successfully.")
	}
}

func discoveryWorker(discoveryJobs chan DiscoveryWork) {
	for job := range discoveryJobs {
		job.retries--
		if job.retries > 0 {
			discoverProcessJVM(job)
		} else {
			log.Debug().Msgf("Process discovery retries for %d exceeded.", job.p.Pid)
			addPidToIgnoredPids(job)
		}
	}
}

func discover(p *process.Process, retries int) {
	discoveryJobs <- DiscoveryWork{p: p, retries: retries}
}

func updatePids() {
	processes, err := process.Processes()
	if err != nil {
		log.Error().Err(err).Msg("Failed to list processes")
		return
	}
	for _, p := range processes {
		createTime, err := p.CreateTime()
		if err != nil {
			log.Err(err).Msgf("Failed to get process creation time. Pid: %d. Error: %s", p.Pid, err.Error())
			return
		}
		cacheId := createCacheId(p.Pid, createTime)
		if utils.ContainsString(ignoredPids, cacheId) {
			log.Trace().Msgf("Process %d is ignored", p.Pid)
			continue
		}
		discover(p, initialRetries)
	}
}

func createCacheId(pid int32, createTime int64) string {
	return fmt.Sprintf("%d-%d", pid, createTime)
}

func discoverProcessJVM(job DiscoveryWork) {
	createTime, err := job.p.CreateTime()
	if err != nil {
		log.Err(err).Msgf("Failed to get process creation time. Pid: %d. Error: %s", job.p.Pid, err.Error())
		return
	}
	cacheId := createCacheId(job.p.Pid, createTime)
	if _, ok := discoveredPids.Load(cacheId); !ok {
		if IsRunning(job.p) {
			if !checkProcessPathAvailable(job.p) {
				log.Debug().Msgf("Process %d is running but path is not available yet.", job.p.Pid)
				go func() {
					time.Sleep(2 * time.Minute)
					discover(job.p, job.retries)
				}()
				return
			}
			success := notifyListenersForNewProcess(job.p)
			if success {
				addPidToDiscoveredPids(job)
			} else {
				//retry
				go func() {
					time.Sleep(1 * time.Minute)
					discover(job.p, job.retries)
				}()
			}
		} else {
			log.Trace().Msgf("Process %d is not running actually", job.p.Pid)
			go func() {
				time.Sleep(1 * time.Minute)
				discover(job.p, job.retries)
			}()
			return
		}
	}
}

func addPidToDiscoveredPids(job DiscoveryWork) {
	createTime, err := job.p.CreateTime()
	if err != nil {
		log.Err(err).Msgf("Failed to get process creation time. Pid: %d. Error: %s", job.p.Pid, err.Error())
		return
	}
	cacheId := createCacheId(job.p.Pid, createTime)
	discoveredPids.Store(cacheId, job.p.Pid)
}

func RemovePidFromDiscoveredPids(pid int32) {
	discoveredPids.Range(func(key, value interface{}) bool {
		if value.(int32) == pid {
			discoveredPids.Delete(key)
			return false
		}
		return true
	})
}

func addPidToIgnoredPids(job DiscoveryWork) {
	createTime, err := job.p.CreateTime()
	if err != nil {
		log.Err(err).Msgf("Failed to get process creation time. Pid: %d. Error: %s", job.p.Pid, err.Error())
		return
	}
	cacheId := createCacheId(job.p.Pid, createTime)
	ignoredPidsMutex.Lock()
	ignoredPids = append(ignoredPids, cacheId)
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
	discoveredPids.Range(func(key, pid interface{}) bool {
		p, err := process.NewProcess(pid.(int32))
		if err != nil {
			log.Warn().Err(err).Msg("Error in listener for notifyListenersForNewProcess")
		} else if isJava(p) {
			listener.NewJavaProcess(p)
		}
		return true
	})
}

func RemoveListener(listener Listener) {
	for i, l := range listeners {
		if l == listener {
			listeners = append(listeners[:i], listeners[i+1:]...)
			break
		}
	}
}

func checkProcessPathAvailable(p *process.Process) bool {
	_, err := GetProcessPath(p)
	return err == nil
}

func isJava(p *process.Process) bool {
	path, err := GetProcessPath(p)
	if err != nil {
		time.Sleep(1 * time.Second)
		path, err = GetProcessPath(p)
		if err != nil {
			return false
		}
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
			log.Debug().Err(err).Msgf("Failed to get process exe, maybe it's not running anymore. Pid: %d. Error: %s", p.Pid, err.Error())
			return "", err
		}
		output = []byte(exe)
	}
	return strings.Trim(string(output), "\n"), nil
}

func IsRunning(p *process.Process) bool {
	status, err := p.Status()
	if err != nil {
		log.Trace().Err(err).Msgf("Failed to get process status. Pid: %d. Error: %s", p.Pid, err.Error())
		return false
	}
	containsString := utils.ContainsString(RunningStates, status)
	if !containsString {
		log.Trace().Msgf("Process %d is not running. Status: %s", p.Pid, status)
	}
	return containsString
}

func IsRunningProcess(pid int32) bool {
	p, err := process.NewProcess(pid)
	if err != nil {
		log.Trace().Err(err).Msg("Process not running anymore")
		return false
	}
	return IsRunning(p)
}
