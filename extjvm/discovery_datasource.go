package extjvm

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/procyon-projects/chrono"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"io"
	"sync"
	"time"
)

var (
	dataSourcePlugin      = utils.GetJarPath("discovery-java-javaagent.jar")
	dataSourceMarkerClass = "javax.sql.DataSource"
)

type DataSourceConnection struct {
	Pid          int32
	DatabaseType string
	JdbcUrl      string
}

type DataSourceApplication struct {
	Pid                   int32
	DataSourceConnections []DataSourceConnection
}

type DataSourceDiscovery struct {
	facade       *jvm.JavaFacade
	applications sync.Map // map[Pid int32]DataSourceApplication
	tasks        sync.Map // map[Pid int32]discoveryTasks
}

func (d *DataSourceDiscovery) JvmAttachedSuccessfully(jvm *jvm.JavaVm) {
	d.scheduleDiscovery(jvm)
}

func (d *DataSourceDiscovery) AttachedProcessStopped(jvm *jvm.JavaVm) {
	d.stopDiscovery(jvm)
	d.applications.Delete(jvm.Pid)
}

func (d *DataSourceDiscovery) getApplications() []DataSourceApplication {
	var result []DataSourceApplication
	d.applications.Range(func(key, value interface{}) bool {
		result = append(result, value.(DataSourceApplication))
		return true
	})
	return result
}

func (d *DataSourceDiscovery) start() {
	d.facade.AddAutoloadAgentPlugin(dataSourcePlugin, dataSourceMarkerClass)
	d.facade.AddAttachedListener(d)
}

func (d *DataSourceDiscovery) stop() {
	d.facade.RemoveAutoloadAgentPlugin(dataSourcePlugin, dataSourceMarkerClass)
}

func (d *DataSourceDiscovery) stopDiscovery(vm *jvm.JavaVm) {
	if holder, ok := d.tasks.Load(vm.Pid); ok {
		if holder.(*discoveryTasks).task30s != nil {
			holder.(*discoveryTasks).task30s.Cancel()
		}
		if holder.(*discoveryTasks).task60s != nil {
			holder.(*discoveryTasks).task60s.Cancel()
		}
		if holder.(*discoveryTasks).task15m != nil {
			holder.(*discoveryTasks).task15m.Cancel()
		}
	}
}

func (d *DataSourceDiscovery) scheduleDiscovery(javaVm *jvm.JavaVm) {
	schedulerHolder := &discoveryTasks{}
	d.tasks.Store(javaVm.Pid, schedulerHolder)

	task30s, err := d.scheduleDiscoveryWithFixedDelay(30*time.Second, javaVm)
	schedulerHolder.task30s = task30s
	if err != nil {
		log.Error().Err(err).Msg("Failed to schedule DataSource Watcher in 30s interval.")
		return
	} else {
		log.Info().Msg("DataSource Watcher Task in 30s interval has been scheduled successfully.")
	}

	go func() {
		time.Sleep(5 * time.Minute)
		task30s.Cancel()
		log.Info().Msg("DataSource Watcher in 30s interval has been canceled.")
		task60s, err := d.scheduleDiscoveryWithFixedDelay(60*time.Second, javaVm)
		schedulerHolder.task60s = task60s
		if err != nil {
			log.Error().Err(err).Msg("Failed to schedule DataSource Watcher in 60s interval for VM Name: " + javaVm.VmName + " and PID: " + string(javaVm.Pid))
			return
		} else {
			log.Info().Msg("DataSource Watcher Task in 60s interval has been scheduled successfully for VM Name: " + javaVm.VmName + " and PID: " + string(javaVm.Pid))
		}
		go func() {
			time.Sleep(5 * time.Minute)
			task60s.Cancel()
			log.Info().Msg("DataSource Watcher in 60s interval has been canceled for VM Name: " + javaVm.VmName + " and PID: " + string(javaVm.Pid))
			task15m, err := d.scheduleDiscoveryWithFixedDelay(15*time.Minute, javaVm)
			schedulerHolder.task15m = task15m
			if err != nil {
				log.Error().Err(err).Msg("Failed to schedule DataSource Watcher in 15m interval for VM Name: " + javaVm.VmName + " and PID: " + string(javaVm.Pid))
				return
			} else {
				log.Info().Msg("DataSource Watcher Task in 15m interval has been scheduled successfully for VM Name: " + javaVm.VmName + " and PID: " + string(javaVm.Pid))
			}
		}()
	}()
}

func (d *DataSourceDiscovery) scheduleDiscoveryWithFixedDelay(interval time.Duration, vm *jvm.JavaVm) (chrono.ScheduledTask, error) {
	taskScheduler := chrono.NewDefaultTaskScheduler()
	return taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		d.dataSourceDiscover(vm)
	}, interval)
}

func (d *DataSourceDiscovery) dataSourceDiscover(javaVm *jvm.JavaVm) {
	if !d.facade.HasAgentPlugin(javaVm, dataSourcePlugin) {
		return
	}

	if dataSourceApplication := d.createDataSourceApplication(javaVm); dataSourceApplication != nil {
		d.applications.Store(javaVm.Pid, *dataSourceApplication)
		log.Info().Msgf("DataSource discovered on PID %d: %+v", javaVm.Pid, dataSourceApplication)
	}
}

func (d *DataSourceDiscovery) createDataSourceApplication(javaVm *jvm.JavaVm) *DataSourceApplication {
	if dataSourceConnections := d.readDataSourceConnections(javaVm); len(dataSourceConnections) > 0 {
		return &DataSourceApplication{
			Pid:                   javaVm.Pid,
			DataSourceConnections: dataSourceConnections,
		}
	}
	return nil
}

func (d *DataSourceDiscovery) readDataSourceConnections(javaVm *jvm.JavaVm) []DataSourceConnection {
	connections, err := d.facade.SendCommandToAgentWithHandler(javaVm, "java-datasource-connection", "", func(response io.Reader) (any, error) {
		var connections []DataSourceConnection
		if err := json.NewDecoder(response).Decode(&connections); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return connections, nil
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read DataSource connections on PID %d", javaVm.Pid)
		return nil
	}

	log.Debug().Msgf("Command '%s:%s' to agent on PID %d returned: %+v", "DataSource-connections", "", javaVm.Pid, connections)
	return connections.([]DataSourceConnection)
}
