package extjvm

import (
	"context"
	"encoding/json"
	"github.com/procyon-projects/chrono"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"sync"
	"time"
)

var (
	//DataSourcePlugin = "javaagent/discovery-java-javaagent.jar"
	DataSourcePlugin       = "/Users/atze/Workspaces/steadybit/repos/agent/agent-bundles-discovery/discovery-java-javaagent/target/discovery-java-javaagent-1.0.0-SNAPSHOT.jar"
	DataSourceMarkerClass  = "javax.sql.DataSource"
	DataSourceApplications = sync.Map{} // map[Pid int32]DataSourceApplication
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

type DataSourceDiscovery struct{}

func (s DataSourceDiscovery) AttachedProcessStopped(jvm *jvm.JavaVm) {
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

func InitDataSourceDiscovery() {
	log.Info().Msg("Init DataSource Plugin")
	AddAutoloadAgentPlugin(DataSourcePlugin, DataSourceMarkerClass)
	AddAttachedListener(DataSourceDiscovery{})
}

func DeactivateDataSourceDiscovery() {
	//TODO: use this method to deactivate the plugin
	RemoveAutoloadAgentPlugin(DataSourcePlugin, DataSourceMarkerClass)
}

func StartDataSourceDiscovery() {
	task30s, err := scheduleDataSourceDiscovery(30 * time.Second)

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
		task60s, err := scheduleDataSourceDiscovery(60 * time.Second)
		if err != nil {
			log.Error().Err(err).Msg("Failed to schedule DataSource Watcher in 60s interval.")
			return
		} else {
			log.Info().Msg("DataSource Watcher Task in 60s interval has been scheduled successfully.")
		}
		go func() {
			time.Sleep(5 * time.Minute)
			task60s.Cancel()
			log.Info().Msg("DataSource Watcher in 60s interval has been canceled.")
			_, err = scheduleDataSourceDiscovery(1 * time.Hour)
			if err != nil {
				log.Error().Err(err).Msg("Failed to schedule DataSource Watcher in 1h interval.")
				return
			} else {
				log.Info().Msg("DataSource Watcher Task in 1h interval has been scheduled successfully.")
			}
		}()

	}()
}

func scheduleDataSourceDiscovery(interval time.Duration) (chrono.ScheduledTask, error) {
	taskScheduler := chrono.NewDefaultTaskScheduler()
	return taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		jvMs := GetJVMs()
		for _, vm := range jvMs {
			DataSourceDiscover(&vm)
		}
	}, interval)
}

func (s DataSourceDiscovery) JvmAttachedSuccessfully(jvm *jvm.JavaVm) {
	DataSourceDiscover(jvm)
}
func DataSourceDiscover(jvm *jvm.JavaVm) {
	if hasDataSourcePlugin(jvm) {
		dataSourceApplication := createDataSource(jvm)
		if dataSourceApplication == nil {
			DataSourceApplications.Store(jvm.Pid, dataSourceApplication)
			log.Trace().Msgf("DataSource discovered on PID %d: %+v", jvm.Pid, dataSourceApplication)
		}
	}
}

func createDataSource(vm *jvm.JavaVm) *DataSourceApplication {
	dataSourceConnections := readDataSourceConnections(vm)
	if dataSourceConnections != nil && len(*dataSourceConnections) > 0 {
		return &DataSourceApplication{
			Pid:                   vm.Pid,
			DataSourceConnections: *dataSourceConnections,
		}
	}
	return nil
}

func readDataSourceConnections(vm *jvm.JavaVm) *[]DataSourceConnection {
	return SendCommandToAgentViaSocket(vm, "java-datasource-connection", "", func(resultMessage string) []DataSourceConnection {
		if resultMessage != "" {
			connections := make([]DataSourceConnection, 0)
			err := json.Unmarshal([]byte(resultMessage), &connections)
			if err != nil {
				log.Trace().Msgf("Command '%s:%s' to agent on PID %s returned error: %s", "DataSource-connections", "", vm.Pid, resultMessage)
				return make([]DataSourceConnection, 0)
			}
			log.Trace().Msgf("Command '%s:%s' to agent on PID %s returned: %s", "DataSource-connections", "", vm.Pid, resultMessage)
			return connections
		} else {
			log.Trace().Msgf("Command '%s:%s' to agent on PID %s returned empty result", "DataSource-connections", "", vm.Pid)
			return make([]DataSourceConnection, 0)
		}
	})
}
func hasDataSourcePlugin(vm *jvm.JavaVm) bool {
	return HasAgentPlugin(vm, DataSourcePlugin)
}
