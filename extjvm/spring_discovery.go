package extjvm

import (
	"github.com/rs/zerolog/log"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
)

var (
	//SpringPlugin = "discovery-springboot-agent/discovery-springboot-javaagent.jar"
	SpringPlugin = "/Users/atze/Workspaces/steadybit/repos/agent/agent-bundles-discovery/discovery-springboot/target/discovery-springboot-agent/discovery-springboot-javaagent.jar"
	SpringMarkerClass                  = "org.springframework.context.ApplicationContext"
	SpringBootMarkerClass              = "org.springframework.boot.ApplicationRunner"
	SpringJdbcTemplateBeanClass        = "org.springframework.jdbc.core.JdbcTemplate"
	SpringResttemplateBeanClass        = "org.springframework.web.client.RestTemplate"
	SpringResttemplateBuilderBeanClass = "org.springframework.boot.web.client.RestTemplateBuilder"
	SpringWebclientBeanClass           = "org.springframework.web.reactive.function.client.WebClient"
	SpringWebclientBuilderBeanClass    = "org.springframework.web.reactive.function.client.WebClient$Builder"

	SpringApplications = make([]SpringApplication, 0) //TODO: make thread safe
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
	MvcMappings        []SpringMvcMapping
	HttpClientRequests []HttpRequest
}

type SpringDiscovery struct{}

func InitSpringDiscovery() {
	log.Info().Msg("Init Spring Plugin")
	AddAutoloadAgentPlugin(SpringPlugin, SpringMarkerClass)
	AddAttachedListener(SpringDiscovery{})
}

func (s SpringDiscovery) JvmAttachedSuccessfully(jvm *jvm.JavaVm) {
	springDiscover(jvm)
}
func springDiscover(jvm *jvm.JavaVm) {
	if hasPlugin(jvm) {
		SpringApplications = append(SpringApplications, createSpringApplication(jvm))
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

func readHttpClientRequest(vm *jvm.JavaVm) []HttpRequest {
	//TODO: implement
	return make([]HttpRequest, 0)
}

func readRequestMappings(vm *jvm.JavaVm) []SpringMvcMapping {
	//TODO: implement
	return make([]SpringMvcMapping, 0)
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
	return *SendCommandToAgentViaSocket(vm, "spring-env", "spring.application.name", func(resultMessage string) string {
		if resultMessage == "" {
			log.Trace().Msgf("Command '%s:%s' to agent on PID %s returned error: %s", "spring-env", "spring.application.name", vm.Pid, resultMessage)
			return ""
		} else {
			log.Trace().Msgf("Command '%s:%s' to agent on PID %s returned : %s", "spring-env", "spring.application.name", vm.Pid, resultMessage)
			return resultMessage
		}
	})
}

func hasPlugin(vm *jvm.JavaVm) bool {
	return HasAgentPlugin(vm, SpringPlugin)
}
