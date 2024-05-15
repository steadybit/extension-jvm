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
	DataSourcePlugin                        = utils.GetJarPath("discovery-java-javaagent.jar")
	DataSourceMarkerClass                   = "javax.sql.DataSource"
	DataSourceApplications                  = sync.Map{} // map[Pid int32]DataSourceApplication
	datasourceVMDiscoverySchedulerHolderMap = sync.Map{} // map[Pid int32]DataSourceDiscoverySchedulerHolder
)

type DataSourceDiscoverySchedulerHolder struct {
	scheduledDatasourceDiscoveryTask30s chrono.ScheduledTask
	scheduledDatasourceDiscoveryTask60s chrono.ScheduledTask
	scheduledDatasourceDiscoveryTask15m chrono.ScheduledTask
}
type DataSourceConnection struct {
	Pid          int32
	DatabaseType string
	JdbcUrl      string
}

type DataSourceApplication struct {
	Pid                   int32
	DataSourceConnections []DataSourceConnection
}

type DataSourceDiscovery struct{}

func (s DataSourceDiscovery) JvmAttachedSuccessfully(jvm *jvm.JavaVm) {
	startScheduledDatasourceDiscovery(jvm)
}
func (s DataSourceDiscovery) AttachedProcessStopped(jvm *jvm.JavaVm) {
	stopScheduledDatasourceDiscoveryForVM(jvm)
	DataSourceApplications.Delete(jvm.Pid)
}

func GetDataSourceApplications() []DataSourceApplication {
	var result []DataSourceApplication
	DataSourceApplications.Range(func(key, value interface{}) bool {
		result = append(result, value.(DataSourceApplication))
		return true
	})
	return result
}

func initDataSourceDiscovery() {
	log.Info().Msg("Init DataSource Plugin")
	addAutoloadAgentPlugin(DataSourcePlugin, DataSourceMarkerClass)
	addAttachedListener(DataSourceDiscovery{})
}
func deactivateDataSourceDiscovery() {
	removeAutoloadAgentPlugin(DataSourcePlugin, DataSourceMarkerClass)
}

func stopScheduledDatasourceDiscoveryForVM(vm *jvm.JavaVm) {
	if holder, ok := datasourceVMDiscoverySchedulerHolderMap.Load(vm.Pid); ok {
		if holder.(*DataSourceDiscoverySchedulerHolder).scheduledDatasourceDiscoveryTask30s != nil {
			holder.(*DataSourceDiscoverySchedulerHolder).scheduledDatasourceDiscoveryTask30s.Cancel()
		}
		if holder.(*DataSourceDiscoverySchedulerHolder).scheduledDatasourceDiscoveryTask60s != nil {
			holder.(*DataSourceDiscoverySchedulerHolder).scheduledDatasourceDiscoveryTask60s.Cancel()
		}
		if holder.(*DataSourceDiscoverySchedulerHolder).scheduledDatasourceDiscoveryTask15m != nil {
			holder.(*DataSourceDiscoverySchedulerHolder).scheduledDatasourceDiscoveryTask15m.Cancel()
		}
	}
}

func startScheduledDatasourceDiscovery(vm *jvm.JavaVm) {
	schedulerHolder := &DataSourceDiscoverySchedulerHolder{}
	datasourceVMDiscoverySchedulerHolderMap.Store(vm.Pid, schedulerHolder)

	task30s, err := scheduleDataSourceDiscoveryForVM(30*time.Second, vm)
	schedulerHolder.scheduledDatasourceDiscoveryTask30s = task30s
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
		task60s, err := scheduleDataSourceDiscoveryForVM(60*time.Second, vm)
		schedulerHolder.scheduledDatasourceDiscoveryTask60s = task60s
		if err != nil {
			log.Error().Err(err).Msg("Failed to schedule DataSource Watcher in 60s interval for VM Name: " + vm.VmName + " and PID: " + string(vm.Pid))
			return
		} else {
			log.Info().Msg("DataSource Watcher Task in 60s interval has been scheduled successfully for VM Name: " + vm.VmName + " and PID: " + string(vm.Pid))
		}
		go func() {
			time.Sleep(5 * time.Minute)
			task60s.Cancel()
			log.Info().Msg("DataSource Watcher in 60s interval has been canceled for VM Name: " + vm.VmName + " and PID: " + string(vm.Pid))
			task15m, err := scheduleDataSourceDiscoveryForVM(15*time.Minute, vm)
			schedulerHolder.scheduledDatasourceDiscoveryTask15m = task15m
			if err != nil {
				log.Error().Err(err).Msg("Failed to schedule DataSource Watcher in 15m interval for VM Name: " + vm.VmName + " and PID: " + string(vm.Pid))
				return
			} else {
				log.Info().Msg("DataSource Watcher Task in 15m interval has been scheduled successfully for VM Name: " + vm.VmName + " and PID: " + string(vm.Pid))
			}
		}()

	}()
}

func scheduleDataSourceDiscoveryForVM(interval time.Duration, vm *jvm.JavaVm) (chrono.ScheduledTask, error) {
	taskScheduler := chrono.NewDefaultTaskScheduler()
	return taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		dataSourceDiscover(vm)
	}, interval)
}

func dataSourceDiscover(jvm *jvm.JavaVm) {
	if !hasAgentPlugin(jvm, DataSourcePlugin) {
		return
	}

	if dataSourceApplication := createDataSourceApplication(jvm); dataSourceApplication != nil {
		DataSourceApplications.Store(jvm.Pid, *dataSourceApplication)
		log.Info().Msgf("DataSource discovered on PID %d: %+v", jvm.Pid, dataSourceApplication)
	}
}

func createDataSourceApplication(vm *jvm.JavaVm) *DataSourceApplication {
	if dataSourceConnections := readDataSourceConnections(vm); len(dataSourceConnections) > 0 {
		return &DataSourceApplication{
			Pid:                   vm.Pid,
			DataSourceConnections: dataSourceConnections,
		}
	}
	return nil
}

func readDataSourceConnections(vm *jvm.JavaVm) []DataSourceConnection {
	connections, err := SendCommandToAgentViaSocket(vm, "java-datasource-connection", "", func(response io.Reader) (*[]DataSourceConnection, error) {
		connections := make([]DataSourceConnection, 0)
		if err := json.NewDecoder(response).Decode(&connections); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &connections, nil
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read DataSource connections on PID %d", vm.Pid)
		return nil
	}

	log.Debug().Msgf("Command '%s:%s' to agent on PID %d returned: %+v", "DataSource-connections", "", vm.Pid, connections)
	return *connections
}
