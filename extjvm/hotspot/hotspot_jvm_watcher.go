package hotspot

import (
	"context"
	"github.com/procyon-projects/chrono"
	"github.com/rs/zerolog/log"
	"github.com/shirou/gopsutil/process"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"time"
)

type Listener interface {
	NewHotspotProcess(p *process.Process)
}

var pids []int32
var listeners []Listener

func Start() {
	taskScheduler := chrono.NewDefaultTaskScheduler()

	_, err := taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		updatePids()
	}, 5*time.Second)

	if err == nil {
		log.Info().Msg("Hotspot JVM Watcher Task has been scheduled successfully.")
	}
}
func updatePids() {
	newPids := GetJvmPids()
	for _, pid := range newPids {
		if !utils.Contains(pids, pid) {
			pids = append(pids, pid)
			p, err := process.NewProcess(pid)
			if err != nil {
				log.Warn().Err(err).Msg("Error in connecting to newProcess")
				continue
			}
			newProcess(p)
		}
	}
}

func newProcess(p *process.Process) {
	log.Trace().Msgf("Discovered new java process: %+v", p)
	for _, listener := range listeners {
		listener.NewHotspotProcess(p)
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
