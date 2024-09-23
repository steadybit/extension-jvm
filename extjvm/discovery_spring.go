// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package extjvm

import (
	"codnect.io/chrono"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"github.com/steadybit/extension-kit/extutil"
	"io"
	"sync"
)

const (
	springMarkerClass                  = "org.springframework.context.ApplicationContext"
	springBootMarkerClass              = "org.springframework.boot.ApplicationRunner"
	springJdbcTemplateBeanClass        = "org.springframework.jdbc.core.JdbcTemplate"
	springRestTemplateBeanClass        = "org.springframework.web.client.RestTemplate"
	springRestTemplateBuilderBeanClass = "org.springframework.boot.web.client.RestTemplateBuilder"
	springWebclientBeanClass           = "org.springframework.web.reactive.function.client.WebClient"
	springWebclientBuilderBeanClass    = "org.springframework.web.reactive.function.client.WebClient$Builder"
)

var (
	springPlugin = utils.GetJarPath("discovery-springboot-javaagent.jar")
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
	facade        jvm.JavaFacade
	taskScheduler chrono.TaskScheduler
	applications  sync.Map // map[Pid int32]SpringApplication
	tasks         sync.Map // map[Pid int32]springVMDiscoverySchedulerHolder
}

func newSpringDiscovery(facade jvm.JavaFacade) *SpringDiscovery {
	return &SpringDiscovery{facade: facade, taskScheduler: chrono.NewDefaultTaskScheduler()}
}

func (d *SpringDiscovery) Attached(jvm jvm.JavaVm) {
	d.scheduleDiscover(jvm)
}
func (d *SpringDiscovery) Detached(jvm jvm.JavaVm) {
	d.cancelDiscover(jvm)
	d.applications.Delete(jvm.Pid())
}

func (d *SpringDiscovery) getApplications() []SpringApplication {
	var result []SpringApplication
	d.applications.Range(func(key, value interface{}) bool {
		result = append(result, value.(SpringApplication))
		return true
	})
	return result
}

func (d *SpringDiscovery) findApplication(pid int32) *SpringApplication {
	for _, application := range d.getApplications() {
		if application.Pid == pid {
			return extutil.Ptr(application)
		}
	}
	return nil
}

func (d *SpringDiscovery) start() {
	d.facade.AddAutoloadAgentPlugin(springPlugin, springMarkerClass)
	d.facade.AddAttachedListener(d)
}

func (d *SpringDiscovery) stop() {
	d.facade.RemoveAttachedListener(d)
	d.facade.RemoveAutoloadAgentPlugin(springPlugin, springMarkerClass)
	<-d.taskScheduler.Shutdown()
	d.tasks = sync.Map{}
}

func (d *SpringDiscovery) cancelDiscover(vm jvm.JavaVm) {
	if t, ok := d.tasks.LoadAndDelete(vm.Pid()); ok {
		t.(*discoveryTask).cancel()
	}
}

func (d *SpringDiscovery) scheduleDiscover(javaVm jvm.JavaVm) {
	t := &discoveryTask{}

	err := t.scheduleOn(d.taskScheduler, func() {
		d.discover(javaVm)
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to schedule Spring discovery for JVM: %s", javaVm.ToInfoString())
	}

	d.tasks.Store(javaVm.Pid(), t)
}

func (d *SpringDiscovery) discover(javaVm jvm.JavaVm) {
	if d.facade.HasAgentPlugin(javaVm, springPlugin) {
		springApplication := d.createSpringApplication(javaVm)
		_, loaded := d.applications.Swap(javaVm.Pid(), springApplication)
		if !loaded {
			log.Debug().Msgf("Spring Instance '%s' on PID %d has been discovered: %+v", springApplication.Name, javaVm.Pid(), springApplication)
		}
	}
}

func (d *SpringDiscovery) createSpringApplication(javaVm jvm.JavaVm) SpringApplication {
	return SpringApplication{
		Name:               d.readSpringApplicationName(javaVm),
		Pid:                javaVm.Pid(),
		SpringBoot:         d.isSpringBootApplication(javaVm),
		UsingJdbcTemplate:  d.hasJdbcTemplate(javaVm),
		UsingHttpClient:    d.hasRestTemplate(javaVm) || d.hasWebClient(javaVm),
		MvcMappings:        d.readRequestMappings(javaVm),
		HttpClientRequests: d.readHttpClientRequest(javaVm),
	}
}

func (d *SpringDiscovery) readHttpClientRequest(javaVm jvm.JavaVm) []HttpRequest {
	requests, err := d.facade.SendCommandToAgentWithHandler(javaVm, "spring-httpclient-requests", "", func(response io.Reader) (any, error) {
		var requests []HttpRequest
		if err := json.NewDecoder(response).Decode(&requests); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return requests, nil
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read HttpClient requests on PID %d", javaVm.Pid())
		return nil
	}
	return requests.([]HttpRequest)
}

func (d *SpringDiscovery) readRequestMappings(javaVm jvm.JavaVm) []SpringMvcMapping {
	mappings, err := d.facade.SendCommandToAgentWithHandler(javaVm, "spring-mvc-mappings", "", func(response io.Reader) (any, error) {
		var mappings []SpringMvcMapping
		if err := json.NewDecoder(response).Decode(&mappings); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)

		}
		return mappings, nil
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read Sping MVC mappings on PID %d", javaVm.Pid())
		return nil
	}
	return mappings.([]SpringMvcMapping)
}

func (d *SpringDiscovery) hasWebClient(jvm jvm.JavaVm) bool {
	return d.hasSpringBean(jvm, springWebclientBeanClass) || d.hasSpringBean(jvm, springWebclientBuilderBeanClass)
}

func (d *SpringDiscovery) hasRestTemplate(jvm jvm.JavaVm) bool {
	return d.hasSpringBean(jvm, springRestTemplateBeanClass) || d.hasSpringBean(jvm, springRestTemplateBuilderBeanClass)
}

func (d *SpringDiscovery) hasJdbcTemplate(jvm jvm.JavaVm) bool {
	return d.hasSpringBean(jvm, springJdbcTemplateBeanClass)
}

func (d *SpringDiscovery) isSpringBootApplication(jvm jvm.JavaVm) bool {
	return d.facade.HasClassLoaded(jvm, springBootMarkerClass)
}

func (d *SpringDiscovery) hasSpringBean(javaVm jvm.JavaVm, beanClass string) bool {
	result, err := d.facade.SendCommandToAgent(javaVm, "spring-bean", beanClass)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to check Spring Bean %s on PID %d", beanClass, javaVm.Pid())
		return false
	}
	return result
}

func (d *SpringDiscovery) readSpringApplicationName(javaVm jvm.JavaVm) string {
	name, err := d.facade.SendCommandToAgentWithHandler(javaVm, "spring-env", "spring.application.name", func(response io.Reader) (any, error) {
		result, err := jvm.GetCleanSocketCommandResult(response)
		log.Debug().Msgf("Result from command spring-env:spring.application.name agent on PID %d: %s", javaVm.Pid(), result)
		return result, err
	})
	if err != nil {
		log.Error().Err(err).Msgf("Failed to read Spring Application Name on PID %d", javaVm.Pid())
		return ""
	}
	return name.(string)
}
