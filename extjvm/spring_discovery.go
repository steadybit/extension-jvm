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

	SpringApplications = sync.Map{} // map[Pid int32]SpringApplication
)

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

func (s SpringDiscovery) AttachedProcessStopped(jvm *jvm.JavaVm) {
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

func StartSpringDiscovery() {
	task30s, err := scheduleSpringDiscovery(30 * time.Second)

	if err != nil {
		log.Error().Err(err).Msg("Failed to schedule Spring Watcher in 30s interval.")
		return
	} else {
		log.Info().Msg("Spring Watcher Task in 30s interval has been scheduled successfully.")
	}

	go func() {
		time.Sleep(5 * time.Minute)
		task30s.Cancel()
		log.Info().Msg("Spring Watcher in 30s interval has been canceled.")
		task60s, err := scheduleSpringDiscovery(60 * time.Second)
		if err != nil {
			log.Error().Err(err).Msg("Failed to schedule Spring Watcher in 60s interval.")
			return
		} else {
			log.Info().Msg("Spring Watcher Task in 60s interval has been scheduled successfully.")
		}
		go func() {
			time.Sleep(5 * time.Minute)
			task60s.Cancel()
			log.Info().Msg("Spring Watcher in 60s interval has been canceled.")
			_, err = scheduleSpringDiscovery(1 * time.Hour)
			if err != nil {
				log.Error().Err(err).Msg("Failed to schedule Spring Watcher in 1h interval.")
				return
			} else {
				log.Info().Msg("Spring Watcher Task in 1h interval has been scheduled successfully.")
			}
		}()

	}()
}

func DeactivateSpringDiscovery() {
	RemoveAutoloadAgentPlugin(SpringPlugin, SpringMarkerClass)
}

func scheduleSpringDiscovery(interval time.Duration) (chrono.ScheduledTask, error) {
	taskScheduler := chrono.NewDefaultTaskScheduler()
	return taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		jvMs := GetJVMs()
		for _, vm := range jvMs {
			springDiscover(&vm)
		}
	}, interval)
}

func (s SpringDiscovery) JvmAttachedSuccessfully(jvm *jvm.JavaVm) {
	springDiscover(jvm)
}
func springDiscover(jvm *jvm.JavaVm) {
	log.Trace().Msgf("Discovering Spring Application on PID %d", jvm.Pid)
	if hasSpringPlugin(jvm) {
		springApplication := createSpringApplication(jvm)
		SpringApplications.Store(jvm.Pid, springApplication)
		log.Trace().Msgf("Spring Application '%s' on PID %d has been discovered: %+v", springApplication.Name, jvm.Pid, springApplication)
	} else {
		log.Trace().Msgf("Application on PID %d is not a Spring Application", jvm.Pid)
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
