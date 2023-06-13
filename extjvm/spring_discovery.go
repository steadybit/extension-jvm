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
	//SpringPlugin = "discovery-springboot-agent/discovery-springboot-javaagent.jar"
	SpringPlugin                       = "/Users/atze/Workspaces/steadybit/repos/agent/agent-bundles-discovery/discovery-springboot/target/discovery-springboot-agent/discovery-springboot-javaagent.jar"
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
	Consumes          []string
	Headers           []string
	Methods           []string
	Params            []string
	Produces          []string
	Patterns          []string
	HandlerClass      string
	HandlerName       string
	HandlerDescriptor string
}
type HttpRequest struct {
	Address        string
	Scheme         string
	Timeout        int
	CircuitBreaker bool
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

	return app
}

func readHttpClientRequest(vm *jvm.JavaVm) *[]HttpRequest {
	return SendCommandToAgentViaSocket(vm, "spring-httpclient-requests", "", func(resultMessage string) []HttpRequest {
		if resultMessage != "" {
			requests := make([]HttpRequest, 0)
			err := json.Unmarshal([]byte(resultMessage), &requests)
			if err != nil {
				log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned error: %s", "spring-httpclient-requests", "", vm.Pid, resultMessage)
				return make([]HttpRequest, 0)
			}
			log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned: %s", "spring-httpclient-requests", "", vm.Pid, resultMessage)
			return requests
		} else {
			log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned empty result", "spring-httpclient-requests", "", vm.Pid)
			return make([]HttpRequest, 0)
		}
	})
}

func readRequestMappings(vm *jvm.JavaVm) *[]SpringMvcMapping {
	return SendCommandToAgentViaSocket(vm, "spring-mvc-mappings", "", func(resultMessage string) []SpringMvcMapping {
		if resultMessage != "" {
			mappings := make([]SpringMvcMapping, 0)
			err := json.Unmarshal([]byte(resultMessage), &mappings)
			if err != nil {
				log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned error: %s", "spring-mvc-mappings", "", vm.Pid, resultMessage)
				return make([]SpringMvcMapping, 0)
			}
			log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned: %s", "spring-mvc-mappings", "", vm.Pid, resultMessage)
			return mappings
		} else {
			log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned empty result", "spring-mvc-mappings", "", vm.Pid)
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
	result := SendCommandToAgentViaSocket(vm, "spring-env", "spring.application.name", func(resultMessage string) string {
		if resultMessage == "" {
			log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned error: %s", "spring-env", "spring.application.name", vm.Pid, resultMessage)
			return ""
		} else {
			log.Trace().Msgf("Command '%s:%s' to agent on PID %d returned : %s", "spring-env", "spring.application.name", vm.Pid, resultMessage)
			return resultMessage
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
