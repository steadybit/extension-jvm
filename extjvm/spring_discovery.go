package extjvm

import (
	"bufio"
	"context"
	"encoding/json"
	"github.com/procyon-projects/chrono"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/common"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-kit/extutil"
	"io"
	"sync"
	"time"
)

var (
	SpringPlugin                       = common.GetJarPath("discovery-springboot-javaagent.jar")
	SpringMarkerClass                  = "org.springframework.context.ApplicationContext"
	SpringBootMarkerClass              = "org.springframework.boot.ApplicationRunner"
	SpringJdbcTemplateBeanClass        = "org.springframework.jdbc.core.JdbcTemplate"
	SpringResttemplateBeanClass        = "org.springframework.web.client.RestTemplate"
	SpringResttemplateBuilderBeanClass = "org.springframework.boot.web.client.RestTemplateBuilder"
	SpringWebclientBeanClass           = "org.springframework.web.reactive.function.client.WebClient"
	SpringWebclientBuilderBeanClass    = "org.springframework.web.reactive.function.client.WebClient$Builder"

	SpringApplications                  = sync.Map{} // map[Pid int32]SpringApplication
	springVMDiscoverySchedulerHolderMap = sync.Map{} // map[Pid int32]SpringVMDiscoverySchedulerHolder
)

type SpringVMDiscoverySchedulerHolder struct {
	scheduledSpringDiscoveryTask30s chrono.ScheduledTask
	scheduledSpringDiscoveryTask60s chrono.ScheduledTask
	scheduledSpringDiscoveryTask60m chrono.ScheduledTask
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
	SpringApplications.Delete(jvm.Pid)
}

func GetSpringApplications() []SpringApplication {
	var result []SpringApplication
	SpringApplications.Range(func(key, value interface{}) bool {
		result = append(result, value.(SpringApplication))
		return true
	})
	return result
}

func FindSpringApplication(pid int32) *SpringApplication {
	applications := GetSpringApplications()
	for _, application := range applications {
		if application.Pid == pid {
			return extutil.Ptr(application)
		}
	}
	return nil
}

func InitSpringDiscovery() {
	log.Info().Msg("Init Spring Plugin")
	AddAutoloadAgentPlugin(SpringPlugin, SpringMarkerClass)
	AddAttachedListener(SpringDiscovery{})
}
func stopScheduledSpringDiscoveryForVM(vm *jvm.JavaVm) {
	springVMDiscoverySchedulerHolder, ok := springVMDiscoverySchedulerHolderMap.Load(vm.Pid)
	if ok {
		if springVMDiscoverySchedulerHolder.(*SpringVMDiscoverySchedulerHolder).scheduledSpringDiscoveryTask30s != nil {
			springVMDiscoverySchedulerHolder.(*SpringVMDiscoverySchedulerHolder).scheduledSpringDiscoveryTask30s.Cancel()
		}
		if springVMDiscoverySchedulerHolder.(*SpringVMDiscoverySchedulerHolder).scheduledSpringDiscoveryTask60s != nil {
			springVMDiscoverySchedulerHolder.(*SpringVMDiscoverySchedulerHolder).scheduledSpringDiscoveryTask60s.Cancel()
		}
		if springVMDiscoverySchedulerHolder.(*SpringVMDiscoverySchedulerHolder).scheduledSpringDiscoveryTask60m != nil {
			springVMDiscoverySchedulerHolder.(*SpringVMDiscoverySchedulerHolder).scheduledSpringDiscoveryTask60m.Cancel()
		}
	}
}

func startScheduledSpringDiscovery(vm *jvm.JavaVm) {
	schedulerHolder := &SpringVMDiscoverySchedulerHolder{}
	springVMDiscoverySchedulerHolderMap.Store(vm.Pid, schedulerHolder)

	task30s, err := scheduleSpringDiscoveryForVM(30*time.Second, vm)
	schedulerHolder.scheduledSpringDiscoveryTask30s = task30s

	if err != nil {
		log.Error().Err(err).Msg("Failed to schedule Spring Watcher in 30s interval for VM Name: " + vm.VmName + " and PID: " + string(vm.Pid))
		return
	} else {
		log.Info().Msg("Spring Watcher Task in 30s interval has been scheduled successfully for VM Name: " + vm.VmName + " and PID: " + string(vm.Pid))
	}

	go func() {
		time.Sleep(5 * time.Minute)
		task30s.Cancel()
		log.Info().Msg("Spring Watcher in 30s interval has been canceled for VM Name: " + vm.VmName + " and PID: " + string(vm.Pid))
		task60s, err := scheduleSpringDiscoveryForVM(60*time.Second, vm)
		schedulerHolder.scheduledSpringDiscoveryTask60s = task60s
		if err != nil {
			log.Error().Err(err).Msg("Failed to schedule Spring Watcher in 60s interval for VM Name: " + vm.VmName + " and PID: " + string(vm.Pid))
			return
		} else {
			log.Info().Msg("Spring Watcher Task in 60s interval has been scheduled successfully for VM Name: " + vm.VmName + " and PID: " + string(vm.Pid))
		}
		go func() {
			time.Sleep(5 * time.Minute)
			task60s.Cancel()
			log.Info().Msg("Spring Watcher in 60s interval has been canceled.")
			task60m, err := scheduleSpringDiscoveryForVM(1*time.Hour, vm)
			schedulerHolder.scheduledSpringDiscoveryTask60m = task60m
			if err != nil {
				log.Error().Err(err).Msg("Failed to schedule Spring Watcher in 1h interval for VM Name: " + vm.VmName + " and PID: " + string(vm.Pid))
				return
			} else {
				log.Info().Msg("Spring Watcher Task in 1h interval has been scheduled successfully for VM Name: " + vm.VmName + " and PID: " + string(vm.Pid))
			}
		}()

	}()
}

func DeactivateSpringDiscovery() {
	RemoveAutoloadAgentPlugin(SpringPlugin, SpringMarkerClass)
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
		SpringApplications.Store(jvm.Pid, springApplication)
		log.Trace().Msgf("Spring Application '%s' on PID %d has been discovered: %+v", springApplication.Name, jvm.Pid, springApplication)
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
	log.Info().Msgf("Spring Application '%s' on PID %d has been discovered: %+v", app.Name, vm.Pid, app)
	return app
}

func readHttpClientRequest(vm *jvm.JavaVm) *[]HttpRequest {
	return SendCommandToAgentViaSocket(vm, "spring-httpclient-requests", "", func(rc string, response io.Reader) []HttpRequest {
		if rc == "OK" {
			requests := make([]HttpRequest, 0)
			err := json.NewDecoder(response).Decode(&requests)
			if err != nil {
				resultMessage, _ := bufio.NewReader(response).ReadString('\n')
				log.Debug().Msgf("Command '%s:%s' to agent on PID %d returned error: %s", "spring-httpclient-requests", "", vm.Pid, resultMessage)
				return make([]HttpRequest, 0)
			}
			log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned: %+v", "spring-httpclient-requests", "", vm.Pid, requests)
			return requests
		} else {
			resultMessage, _ := bufio.NewReader(response).ReadString('\n')
			log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned error: %s", "spring-httpclient-requests", "", vm.Pid, resultMessage)
			return make([]HttpRequest, 0)
		}
	})
}

func readRequestMappings(vm *jvm.JavaVm) *[]SpringMvcMapping {
	return SendCommandToAgentViaSocket(vm, "spring-mvc-mappings", "", func(rc string, response io.Reader) []SpringMvcMapping {
		if rc == "OK" {
			mappings := make([]SpringMvcMapping, 0)
			err := json.NewDecoder(response).Decode(&mappings)
			if err != nil {
				resultMessage, _ := bufio.NewReader(response).ReadString('\n')
				log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned error: %s", "spring-mvc-mappings", "", vm.Pid, resultMessage)
				return make([]SpringMvcMapping, 0)
			}
			log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned: %+v", "spring-mvc-mappings", "", vm.Pid, mappings)
			return mappings
		} else {
			resultMessage, _ := bufio.NewReader(response).ReadString('\n')
			log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned error: %s", "spring-mvc-mappings", "", vm.Pid, resultMessage)
			return make([]SpringMvcMapping, 0)
		}
	})
}

func hasWebClient(vm *jvm.JavaVm) bool {
	return SendCommandToAgent(vm, "spring-bean", SpringWebclientBeanClass) || SendCommandToAgent(vm,
		"spring-bean", SpringWebclientBuilderBeanClass)
}

func hasRestTemplate(vm *jvm.JavaVm) bool {
	return SendCommandToAgent(vm, "spring-bean", SpringResttemplateBeanClass) || SendCommandToAgent(
		vm, "spring-bean", SpringResttemplateBuilderBeanClass)
}

func hasJdbcTemplate(vm *jvm.JavaVm) bool {
	return SendCommandToAgent(vm, "spring-bean", SpringJdbcTemplateBeanClass)
}

func isSpringBootApplication(vm *jvm.JavaVm) bool {
	return HasClassLoaded(vm, SpringBootMarkerClass)
}

func readSpringApplicationName(vm *jvm.JavaVm) string {
	result := SendCommandToAgentViaSocket(vm, "spring-env", "spring.application.name", func(rc string, response io.Reader) string {
		if rc == "OK" {
			resultMessage, _ := GetCleanSocketCommandResult(response)
			if resultMessage == "" {
				log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned error: %s", "spring-env", "spring.application.name", vm.Pid, resultMessage)
				return ""
			} else {
				log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned : %s", "spring-env", "spring.application.name", vm.Pid, resultMessage)
				return resultMessage
			}
		} else {
			log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned error: %s", "spring-env", "spring.application.name", vm.Pid, rc)
			return ""
		}

	})
	if result == nil {
		return ""
	}
	return *result
}

func hasSpringPlugin(vm *jvm.JavaVm) bool {
	return HasAgentPlugin(vm, SpringPlugin)
}
