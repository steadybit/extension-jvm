package extjvm

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/procyon-projects/chrono"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"github.com/steadybit/extension-kit/extutil"
	"io"
	"strconv"
	"sync"
	"time"
)

var (
	springPlugin                       = utils.GetJarPath("discovery-springboot2-javaagent.jar")
	springMarkerClass                  = "org.springframework.context.ApplicationContext"
	springBootMarkerClass              = "org.springframework.boot.ApplicationRunner"
	springJdbcTemplateBeanClass        = "org.springframework.jdbc.core.JdbcTemplate"
	springRestTemplateBeanClass        = "org.springframework.web.client.RestTemplate"
	springResttemplateBuilderBeanClass = "org.springframework.boot.web.client.RestTemplateBuilder"
	springWebclientBeanClass           = "org.springframework.web.reactive.function.client.WebClient"
	springWebclientBuilderBeanClass    = "org.springframework.web.reactive.function.client.WebClient$Builder"

	springApplications                  = sync.Map{} // map[Pid int32]SpringApplication
	springVMDiscoverySchedulerHolderMap = sync.Map{} // map[Pid int32]springVMDiscoverySchedulerHolder
)

type springVMDiscoverySchedulerHolder struct {
	scheduledSpringDiscoveryTask30s chrono.ScheduledTask
	scheduledSpringDiscoveryTask60s chrono.ScheduledTask
	scheduledSpringDiscoveryTask15m chrono.ScheduledTask
}

type SpringMvcMapping struct {
	Consumes          []string `json:"consumes"`
	Headers           []string `json:"headers"`
	Methods           []string `json:"methods"`
	Params            []string `json:"params"`
	Produces          []string `json:"produces"`
	Patterns          []string `json:"patterns"`
	HandlerClass      string   `json:"handlerClass"`
	HandlerName       string   `json:"handlerName"`
	HandlerDescriptor string   `json:"handlerDescriptor"`
}

type HttpRequest struct {
	Address        string `json:"address"`
	Scheme         string `json:"scheme"`
	Timeout        int    `json:"timeout"`
	CircuitBreaker bool   `json:"circuitBreaker"`
}

type SpringApplication struct {
	Name               string
	Pid                int32
	SpringBoot         bool
	UsingJdbcTemplate  bool
	UsingHttpClient    bool
	MvcMappings        *[]SpringMvcMapping
	HttpClientRequests *[]HttpRequest
}

type SpringDiscovery struct{}

func (s SpringDiscovery) JvmAttachedSuccessfully(jvm *jvm.JavaVm) {
	startScheduledSpringDiscovery(jvm)
}
func (s SpringDiscovery) AttachedProcessStopped(jvm *jvm.JavaVm) {
	stopScheduledSpringDiscoveryForVM(jvm)
	springApplications.Delete(jvm.Pid)
}

func GetSpringApplications() []SpringApplication {
	var result []SpringApplication
	springApplications.Range(func(key, value interface{}) bool {
		result = append(result, value.(SpringApplication))
		return true
	})
	return result
}

func findSpringApplication(pid int32) *SpringApplication {
	applications := GetSpringApplications()
	for _, application := range applications {
		if application.Pid == pid {
			return extutil.Ptr(application)
		}
	}
	return nil
}

func initSpringDiscovery() {
	log.Info().Msg("Init Spring Plugin")
	addAutoloadAgentPlugin(springPlugin, springMarkerClass)
	addAttachedListener(SpringDiscovery{})
}

func stopScheduledSpringDiscoveryForVM(vm *jvm.JavaVm) {
	if holder, ok := springVMDiscoverySchedulerHolderMap.Load(vm.Pid); ok {
		if holder.(*springVMDiscoverySchedulerHolder).scheduledSpringDiscoveryTask30s != nil {
			holder.(*springVMDiscoverySchedulerHolder).scheduledSpringDiscoveryTask30s.Cancel()
		}
		if holder.(*springVMDiscoverySchedulerHolder).scheduledSpringDiscoveryTask60s != nil {
			holder.(*springVMDiscoverySchedulerHolder).scheduledSpringDiscoveryTask60s.Cancel()
		}
		if holder.(*springVMDiscoverySchedulerHolder).scheduledSpringDiscoveryTask15m != nil {
			holder.(*springVMDiscoverySchedulerHolder).scheduledSpringDiscoveryTask15m.Cancel()
		}
	}
}

func startScheduledSpringDiscovery(vm *jvm.JavaVm) {
	schedulerHolder := &springVMDiscoverySchedulerHolder{}
	springVMDiscoverySchedulerHolderMap.Store(vm.Pid, schedulerHolder)

	task30s, err := scheduleSpringDiscoveryForVM(30*time.Second, vm)
	schedulerHolder.scheduledSpringDiscoveryTask30s = task30s

	if err != nil {
		log.Error().Err(err).Msg("Failed to schedule Spring Watcher in 30s interval for VM Name: " + vm.VmName + " and PID: " + strconv.Itoa(int(vm.Pid)))
		return
	} else {
		log.Info().Msg("Spring Watcher Task in 30s interval has been scheduled successfully for VM Name: " + vm.VmName + " and PID: " + strconv.Itoa(int(vm.Pid)))
	}

	go func() {
		time.Sleep(5 * time.Minute)
		task30s.Cancel()
		log.Info().Msg("Spring Watcher in 30s interval has been canceled for VM Name: " + vm.VmName + " and PID: " + strconv.Itoa(int(vm.Pid)))
		task60s, err := scheduleSpringDiscoveryForVM(60*time.Second, vm)
		schedulerHolder.scheduledSpringDiscoveryTask60s = task60s
		if err != nil {
			log.Error().Err(err).Msg("Failed to schedule Spring Watcher in 60s interval for VM Name: " + vm.VmName + " and PID: " + strconv.Itoa(int(vm.Pid)))
			return
		} else {
			log.Info().Msg("Spring Watcher Task in 60s interval has been scheduled successfully for VM Name: " + vm.VmName + " and PID: " + strconv.Itoa(int(vm.Pid)))
		}
		go func() {
			time.Sleep(5 * time.Minute)
			task60s.Cancel()
			log.Info().Msg("Spring Watcher in 60s interval has been canceled.")
			task15m, err := scheduleSpringDiscoveryForVM(15*time.Minute, vm)
			schedulerHolder.scheduledSpringDiscoveryTask15m = task15m
			if err != nil {
				log.Error().Err(err).Msg("Failed to schedule Spring Watcher in 15m interval for VM Name: " + vm.VmName + " and PID: " + strconv.Itoa(int(vm.Pid)))
				return
			} else {
				log.Info().Msg("Spring Watcher Task in 15m interval has been scheduled successfully for VM Name: " + vm.VmName + " and PID: " + strconv.Itoa(int(vm.Pid)))
			}
		}()

	}()
}

func deactivateSpringDiscovery() {
	removeAutoloadAgentPlugin(springPlugin, springMarkerClass)
}
func scheduleSpringDiscoveryForVM(interval time.Duration, vm *jvm.JavaVm) (chrono.ScheduledTask, error) {
	taskScheduler := chrono.NewDefaultTaskScheduler()
	return taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		springDiscover(vm)
	}, interval)
}

func springDiscover(jvm *jvm.JavaVm) {
	if hasSpringPlugin(jvm) {
		springApplication := createSpringApplication(jvm)
		springApplications.Store(jvm.Pid, springApplication)
		log.Trace().Msgf("Spring Instance '%s' on PID %d has been discovered: %+v", springApplication.Name, jvm.Pid, springApplication)
	}
}

func createSpringApplication(vm *jvm.JavaVm) SpringApplication {
	app := SpringApplication{
		Name:               readSpringApplicationName(vm),
		Pid:                vm.Pid,
		SpringBoot:         isSpringBootApplication(vm),
		UsingJdbcTemplate:  hasJdbcTemplate(vm),
		UsingHttpClient:    hasRestTemplate(vm) || hasWebClient(vm),
		MvcMappings:        readRequestMappings(vm),
		HttpClientRequests: readHttpClientRequest(vm),
	}
	log.Info().Msgf("Spring Instance '%s' on PID %d has been discovered: %+v", app.Name, vm.Pid, app)
	return app
}

func readHttpClientRequest(vm *jvm.JavaVm) *[]HttpRequest {
	requests, err := SendCommandToAgentViaSocket(vm, "spring-httpclient-requests", "", func(response io.Reader) (*[]HttpRequest, error) {
		requests := make([]HttpRequest, 0)
		if err := json.NewDecoder(response).Decode(&requests); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &requests, nil
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read HttpClient requests on PID %d", vm.Pid)
		return nil
	}
	return requests
}

func readRequestMappings(vm *jvm.JavaVm) *[]SpringMvcMapping {
	mappings, err := SendCommandToAgentViaSocket(vm, "spring-mvc-mappings", "", func(response io.Reader) (*[]SpringMvcMapping, error) {
		mappings := make([]SpringMvcMapping, 0)
		if err := json.NewDecoder(response).Decode(&mappings); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)

		}
		return &mappings, nil
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read Sping MVC mappings on PID %d", vm.Pid)
		return nil
	}
	return mappings
}

func hasWebClient(vm *jvm.JavaVm) bool {
	return SendCommandToAgent(vm, "spring-bean", springWebclientBeanClass) || SendCommandToAgent(vm, "spring-bean", springWebclientBuilderBeanClass)
}

func hasRestTemplate(vm *jvm.JavaVm) bool {
	return SendCommandToAgent(vm, "spring-bean", springRestTemplateBeanClass) || SendCommandToAgent(vm, "spring-bean", springResttemplateBuilderBeanClass)
}

func hasJdbcTemplate(vm *jvm.JavaVm) bool {
	return SendCommandToAgent(vm, "spring-bean", springJdbcTemplateBeanClass)
}

func isSpringBootApplication(vm *jvm.JavaVm) bool {
	return hasClassLoaded(vm, springBootMarkerClass)
}

func readSpringApplicationName(vm *jvm.JavaVm) string {
	name, err := SendCommandToAgentViaSocket(vm, "spring-env", "spring.application.name", func(response io.Reader) (*string, error) {
		s, err := GetCleanSocketCommandResult(response)
		return &s, err
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read Spring Application Name on PID %d", vm.Pid)
		return ""
	}
	return *name
}

func hasSpringPlugin(vm *jvm.JavaVm) bool {
	return hasAgentPlugin(vm, springPlugin)
}
