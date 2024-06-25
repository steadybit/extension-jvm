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
	springRestTemplateBuilderBeanClass = "org.springframework.boot.web.client.RestTemplateBuilder"
	springWebclientBeanClass           = "org.springframework.web.reactive.function.client.WebClient"
	springWebclientBuilderBeanClass    = "org.springframework.web.reactive.function.client.WebClient$Builder"
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
	MvcMappings        []SpringMvcMapping
	HttpClientRequests []HttpRequest
}

type SpringDiscovery struct {
	facade       *jvm.JavaFacade
	applications sync.Map // map[Pid int32]SpringApplication
	tasks        sync.Map // map[Pid int32]springVMDiscoverySchedulerHolder
}

func (s *SpringDiscovery) JvmAttachedSuccessfully(jvm *jvm.JavaVm) {
	s.scheduleDiscovery(jvm)
}
func (s *SpringDiscovery) AttachedProcessStopped(jvm *jvm.JavaVm) {
	s.stopDiscovery(jvm)
	s.applications.Delete(jvm.Pid)
}

func (s *SpringDiscovery) GetApplications() []SpringApplication {
	var result []SpringApplication
	s.applications.Range(func(key, value interface{}) bool {
		result = append(result, value.(SpringApplication))
		return true
	})
	return result
}

func (s *SpringDiscovery) FindSpringApplication(pid int32) *SpringApplication {
	for _, application := range s.GetApplications() {
		if application.Pid == pid {
			return extutil.Ptr(application)
		}
	}
	return nil
}

func (s *SpringDiscovery) start() {
	s.facade.AddAutoloadAgentPlugin(springPlugin, springMarkerClass)
	s.facade.AddAttachedListener(s)
}

func (d *SpringDiscovery) stop() {
	d.facade.RemoveAutoloadAgentPlugin(springPlugin, springMarkerClass)
}

func (s *SpringDiscovery) stopDiscovery(vm *jvm.JavaVm) {
	if holder, ok := s.tasks.Load(vm.Pid); ok {
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

func (s *SpringDiscovery) scheduleDiscovery(vm *jvm.JavaVm) {
	schedulerHolder := &discoveryTasks{}
	s.tasks.Store(vm.Pid, schedulerHolder)

	task30s, err := s.scheduleDiscoveryWithFixedDelay(30*time.Second, vm)
	schedulerHolder.task30s = task30s

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
		task60s, err := s.scheduleDiscoveryWithFixedDelay(60*time.Second, vm)
		schedulerHolder.task60s = task60s
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
			task15m, err := s.scheduleDiscoveryWithFixedDelay(15*time.Minute, vm)
			schedulerHolder.task15m = task15m
			if err != nil {
				log.Error().Err(err).Msg("Failed to schedule Spring Watcher in 15m interval for VM Name: " + vm.VmName + " and PID: " + strconv.Itoa(int(vm.Pid)))
				return
			} else {
				log.Info().Msg("Spring Watcher Task in 15m interval has been scheduled successfully for VM Name: " + vm.VmName + " and PID: " + strconv.Itoa(int(vm.Pid)))
			}
		}()

	}()
}

func (s *SpringDiscovery) deactivateSpringDiscovery() {
	s.facade.RemoveAutoloadAgentPlugin(springPlugin, springMarkerClass)
}
func (s *SpringDiscovery) scheduleDiscoveryWithFixedDelay(interval time.Duration, vm *jvm.JavaVm) (chrono.ScheduledTask, error) {
	taskScheduler := chrono.NewDefaultTaskScheduler()
	return taskScheduler.ScheduleWithFixedDelay(func(ctx context.Context) {
		s.springDiscover(vm)
	}, interval)
}

func (s *SpringDiscovery) springDiscover(javaVm *jvm.JavaVm) {
	if s.facade.HasAgentPlugin(javaVm, springPlugin) {
		springApplication := s.createSpringApplication(javaVm)
		s.applications.Store(javaVm.Pid, springApplication)
		log.Trace().Msgf("Spring Instance '%s' on PID %d has been discovered: %+v", springApplication.Name, javaVm.Pid, springApplication)
	}
}

func (s *SpringDiscovery) createSpringApplication(javaVm *jvm.JavaVm) SpringApplication {
	app := SpringApplication{
		Name:               s.readSpringApplicationName(javaVm),
		Pid:                javaVm.Pid,
		SpringBoot:         s.isSpringBootApplication(javaVm),
		UsingJdbcTemplate:  s.hasJdbcTemplate(javaVm),
		UsingHttpClient:    s.hasRestTemplate(javaVm) || s.hasWebClient(javaVm),
		MvcMappings:        s.readRequestMappings(javaVm),
		HttpClientRequests: s.readHttpClientRequest(javaVm),
	}
	log.Info().Msgf("Spring Instance '%s' on PID %d has been discovered: %+v", app.Name, javaVm.Pid, app)
	return app
}

func (s *SpringDiscovery) readHttpClientRequest(javaVm *jvm.JavaVm) []HttpRequest {
	requests, err := s.facade.SendCommandToAgentWithHandler(javaVm, "spring-httpclient-requests", "", func(response io.Reader) (any, error) {
		var requests []HttpRequest
		if err := json.NewDecoder(response).Decode(&requests); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return requests, nil
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read HttpClient requests on PID %d", javaVm.Pid)
		return nil
	}
	return requests.([]HttpRequest)
}

func (s *SpringDiscovery) readRequestMappings(javaVm *jvm.JavaVm) []SpringMvcMapping {
	mappings, err := s.facade.SendCommandToAgentWithHandler(javaVm, "spring-mvc-mappings", "", func(response io.Reader) (any, error) {
		var mappings []SpringMvcMapping
		if err := json.NewDecoder(response).Decode(&mappings); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)

		}
		return mappings, nil
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read Sping MVC mappings on PID %d", javaVm.Pid)
		return nil
	}
	return mappings.([]SpringMvcMapping)
}

func (s *SpringDiscovery) hasWebClient(javaVm *jvm.JavaVm) bool {
	return s.hasSpringBean(javaVm, springWebclientBeanClass) || s.hasSpringBean(javaVm, springWebclientBuilderBeanClass)
}

func (s *SpringDiscovery) hasRestTemplate(javaVm *jvm.JavaVm) bool {
	return s.hasSpringBean(javaVm, springRestTemplateBeanClass) || s.hasSpringBean(javaVm, springRestTemplateBuilderBeanClass)
}

func (s *SpringDiscovery) hasJdbcTemplate(javaVm *jvm.JavaVm) bool {
	return s.hasSpringBean(javaVm, springJdbcTemplateBeanClass)
}

func (s *SpringDiscovery) isSpringBootApplication(vm *jvm.JavaVm) bool {
	return s.facade.HasClassLoaded(vm, springBootMarkerClass)
}

func (s *SpringDiscovery) hasSpringBean(javaVm *jvm.JavaVm, beanClass string) bool {
	result, err := s.facade.SendCommandToAgent(javaVm, "spring-bean", beanClass)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to check Spring Bean %s on PID %d", beanClass, javaVm.Pid)
		return false
	}
	return result
}

func (s *SpringDiscovery) readSpringApplicationName(javaVm *jvm.JavaVm) string {
	name, err := s.facade.SendCommandToAgentWithHandler(javaVm, "spring-env", "spring.application.name", func(response io.Reader) (any, error) {
		s, err := jvm.GetCleanSocketCommandResult(response)
		return s, err
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read Spring Application Name on PID %d", javaVm.Pid)
		return ""
	}
	return name.(string)
}
