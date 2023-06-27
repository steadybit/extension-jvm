package hotspot

import (
	"context"
	"github.com/procyon-projects/chrono"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/process"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"sync"
	"time"
)

type Listener interface {
	NewHotspotProcess(p *process.Process) bool
}

type DiscoveryWork struct {
	pid     int32
	retries int
}

var (
	hotspotPidsMutex sync.Mutex
	hotspotPids      []int32
	listeners        []Listener

	hotspotDiscoveryJobs = make(chan DiscoveryWork)
)

const (
	initialRetries = 3
)

func Start() {
	taskScheduler := chrono.NewDefaultTaskScheduler()

	_, err := taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		updatePids()
	}, 5*time.Second)

	if err == nil {
		log.Info().Msg("Hotspot JVM Watcher Task has been scheduled successfully.")
	}

	// create hotspot discovery worker pool
	for w := 1; w <= 4; w++ {
		go discoveryWorker(hotspotDiscoveryJobs)
	}
}

func discoveryWorker(hotspotDiscoveryJobs chan DiscoveryWork) {
	for job := range hotspotDiscoveryJobs {
		job.retries--
		if job.retries > 0 {
			discoverHotspotJvm(job)
		} else {
			log.Warn().Msgf("Hotspot discovery retries for %d exceeded.", job.pid)
		}
	}
}

func discover(pid int32, retries int) {
	if retries != initialRetries {
		time.Sleep(1 * time.Second)
	}
	hotspotDiscoveryJobs <- DiscoveryWork{pid: pid, retries: retries}
}
func updatePids() {
	newPids := GetJvmPids()
	for _, pid := range newPids {
		discover(pid, initialRetries)
	}
}

func discoverHotspotJvm(work DiscoveryWork) {
	if !utils.Contains(hotspotPids, work.pid) {
		p, err := process.NewProcess(work.pid)
		if err != nil {
			log.Warn().Err(err).Msg("Error in connecting to newProcess")
			//retry
			discover(work.pid, work.retries)
			return
		}
		success := newHotspotProcess(p)
		if success {
			addToDiscoveredHotspotPids(work.pid)
		} else {
			//retry
			discover(work.pid, work.retries)
		}
	}
}

func addToDiscoveredHotspotPids(pid int32) {
	hotspotPidsMutex.Lock()
	hotspotPids = append(hotspotPids, pid)
	hotspotPidsMutex.Unlock()
}

func newHotspotProcess(p *process.Process) bool {
	log.Trace().Msgf("Discovered new java process: %+v", p)
	success := true
	for _, listener := range listeners {
		lastResult := listener.NewHotspotProcess(p)
		if !lastResult {
			success = false
		}
	}
	return success
}

func AddListener(listener Listener) {
	listeners = append(listeners, listener)
	for _, pid := range hotspotPids {
		p, err := process.NewProcess(pid)
		if err != nil {
			log.Warn().Err(err).Msg("Error in listener for newProcess")
			continue
		}
		listener.NewHotspotProcess(p)
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
