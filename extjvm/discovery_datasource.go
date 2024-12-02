package extjvm

import (
	"codnect.io/chrono"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"io"
	"sync"
)

const (
	dataSourceMarkerClass = "javax.sql.DataSource"
)

var (
	dataSourcePlugin = utils.GetJarPath("discovery-java-javaagent.jar")
)

type dataSourceConnection struct {
	Pid          int32
	DatabaseType string
	JdbcUrl      string
}

type dataSourceApplication struct {
	Pid                   int32
	DataSourceConnections []dataSourceConnection
}

type DataSourceDiscovery struct {
	facade        jvm.JavaFacade
	taskScheduler chrono.TaskScheduler
	applications  sync.Map // map[Pid int32]dataSourceApplication
	tasks         sync.Map // map[Pid int32]discoveryTask
}

func newDataSourceDiscovery(facade jvm.JavaFacade) *DataSourceDiscovery {
	return &DataSourceDiscovery{facade: facade, taskScheduler: chrono.NewDefaultTaskScheduler()}
}

func (d *DataSourceDiscovery) Attached(jvm jvm.JavaVm) {
	d.scheduleDiscover(jvm)
}

func (d *DataSourceDiscovery) Detached(jvm jvm.JavaVm) {
	d.cancelDiscover(jvm)
	d.applications.Delete(jvm.Pid())
}

func (d *DataSourceDiscovery) getApplications() []dataSourceApplication {
	var result []dataSourceApplication
	d.applications.Range(func(key, value interface{}) bool {
		result = append(result, value.(dataSourceApplication))
		return true
	})
	return result
}

func (d *DataSourceDiscovery) start() {
	d.facade.AddAutoloadAgentPlugin(dataSourcePlugin, dataSourceMarkerClass)
	d.facade.AddAttachedListener(d)
}

func (d *DataSourceDiscovery) stop() {
	d.facade.RemoveAttachedListener(d)
	d.facade.RemoveAutoloadAgentPlugin(dataSourcePlugin, dataSourceMarkerClass)
	<-d.taskScheduler.Shutdown()
	d.tasks = sync.Map{}
}

func (d *DataSourceDiscovery) cancelDiscover(vm jvm.JavaVm) {
	if t, ok := d.tasks.LoadAndDelete(vm.Pid()); ok {
		t.(*discoveryTask).cancel()
	}
}

func (d *DataSourceDiscovery) scheduleDiscover(javaVm jvm.JavaVm) {
	t := &discoveryTask{}

	err := t.scheduleOn(d.taskScheduler, func() {
		d.discover(javaVm)
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to schedule DataSource discovery for JVM: %s", javaVm.ToInfoString())
	}

	d.tasks.Store(javaVm.Pid(), t)
}

func (d *DataSourceDiscovery) discover(javaVm jvm.JavaVm) {
	if !d.facade.HasAgentPlugin(javaVm, dataSourcePlugin) {
		return
	}

	if dataSourceApplication := d.createDataSourceApplication(javaVm); dataSourceApplication != nil {
		_, loaded := d.applications.Swap(javaVm.Pid(), *dataSourceApplication)
		if !loaded {
			log.Debug().Msgf("DataSource discovered on PID %d: %+v", javaVm.Pid(), dataSourceApplication)
		}
	}
}

func (d *DataSourceDiscovery) createDataSourceApplication(javaVm jvm.JavaVm) *dataSourceApplication {
	if dataSourceConnections := d.readDataSourceConnections(javaVm); len(dataSourceConnections) > 0 {
		return &dataSourceApplication{
			Pid:                   javaVm.Pid(),
			DataSourceConnections: dataSourceConnections,
		}
	}
	return nil
}

func (d *DataSourceDiscovery) readDataSourceConnections(javaVm jvm.JavaVm) []dataSourceConnection {
	connections, err := d.facade.SendCommandToAgentWithHandler(javaVm, "java-datasource-connection", "", func(response io.Reader) (any, error) {
		var connections []dataSourceConnection
		if err := json.NewDecoder(response).Decode(&connections); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return connections, nil
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read DataSource connections on PID %d", javaVm.Pid())
		return nil
	}

	log.Debug().Msgf("Command '%s:%s' to agent on PID %d returned: %+v", "DataSource-connections", "", javaVm.Pid(), connections)
	return connections.([]dataSourceConnection)
}
