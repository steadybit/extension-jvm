/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extjvm

import (
	"context"
	"fmt"
	"github.com/procyon-projects/chrono"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-jvm/config"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"golang.org/x/sys/unix"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type jvmDiscovery struct {
	jvms       jvmLister
	datasource *DataSourceDiscovery
	spring     *SpringDiscovery
}

var (
	_ discovery_kit_sdk.TargetDescriber          = (*jvmDiscovery)(nil)
	_ discovery_kit_sdk.AttributeDescriber       = (*jvmDiscovery)(nil)
	_ discovery_kit_sdk.EnrichmentRulesDescriber = (*jvmDiscovery)(nil)

	discoverySchedule = []time.Duration{
		0 * time.Second,
		30 * time.Second,
		1 * time.Minute,
		5 * time.Minute,
	}
)

type discoveryTask struct {
	task  chrono.ScheduledTask
	count int
}

func (t *discoveryTask) nextDelay() time.Duration {
	if t.count >= len(discoverySchedule) {
		return discoverySchedule[len(discoverySchedule)-1]
	} else {
		return discoverySchedule[t.count]
	}
}

func (t *discoveryTask) cancel() {
	if t.task != nil {
		t.task.Cancel()
	}
}

func (t *discoveryTask) scheduleOn(scheduler chrono.TaskScheduler, f func()) (err error) {
	t.task, err = scheduler.Schedule(func(ctx context.Context) {
		f()

		t.count++
		if err := t.scheduleOn(scheduler, f); err != nil {
			log.Warn().Err(err).Msg("Failed to schedule next task")
		}
	}, chrono.WithTime(time.Now().Add(t.nextDelay())))
	return
}

type jvmLister interface {
	GetJvms() []jvm.JavaVm
}

func NewJvmDiscovery(jvms jvmLister, datasource *DataSourceDiscovery, spring *SpringDiscovery) discovery_kit_sdk.TargetDiscovery {
	discovery := &jvmDiscovery{
		jvms:       jvms,
		datasource: datasource,
		spring:     spring,
	}
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsInterval(context.Background(), 30*time.Second),
	)
}

func StartJvmInfrastructure() (jvm.JavaFacade, *DataSourceDiscovery, *SpringDiscovery) {
	facade := jvm.NewJavaFacade()
	datasource := newDataSourceDiscovery(facade)
	spring := newSpringDiscovery(facade)

	installSignalHandler(facade, datasource, spring)

	facade.Start()
	datasource.start()
	spring.start()

	return facade, datasource, spring
}

func installSignalHandler(facade jvm.JavaFacade, datasource *DataSourceDiscovery, spring *SpringDiscovery) {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)
	go func(signals <-chan os.Signal) {
		for s := range signals {
			signalName := unix.SignalName(s.(syscall.Signal))

			log.Info().Str("signal", signalName).Msg("received signal - stopping all active discoveries")
			datasource.stop()
			spring.stop()
			facade.Stop()
		}
	}(signalChannel)
}

func (j *jvmDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id: targetID,
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			CallInterval: extutil.Ptr(config.Config.DiscoveryCallInterval),
		},
	}
}

func (j *jvmDiscovery) DescribeTarget() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:      targetID,
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		Icon:    extutil.Ptr(targetIcon),

		// Labels used in the UI
		Label: discovery_kit_api.PluralLabel{One: "JVM instance", Other: "JVM instances"},

		// Category for the targets to appear in
		Category: extutil.Ptr(category),

		// Specify attributes shown in table columns and to be used for sorting
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "jvm-instance.name"},
				{Attribute: "instance.type"},
				{Attribute: "k8s.namespace"},
				{Attribute: "k8s.cluster-name"},
				{Attribute: "k8s.deployment"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "jvm-instance.name",
					Direction: "ASC",
				},
			},
		},
	}
}

func (j *jvmDiscovery) DescribeEnrichmentRules() []discovery_kit_api.TargetEnrichmentRule {
	return []discovery_kit_api.TargetEnrichmentRule{
		getKubernetesContainerToJvmEnrichmentRule(),
		getContainerToJvmEnrichmentRule(),
		getJvmToContainerEnrichmentRule(),
	}
}

func getKubernetesContainerToJvmEnrichmentRule() discovery_kit_api.TargetEnrichmentRule {
	return discovery_kit_api.TargetEnrichmentRule{
		Id:      "com.steadybit.extension_jvm.k8s-container-to-jvm",
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		Src: discovery_kit_api.SourceOrDestination{
			Type: "com.steadybit.extension_kubernetes.kubernetes-container",
			Selector: map[string]string{
				"k8s.container.id.stripped": "${dest.container.id.stripped}",
			},
		},
		Dest: discovery_kit_api.SourceOrDestination{
			Type: targetID,
			Selector: map[string]string{
				"container.id.stripped": "${src.k8s.container.id.stripped}",
			},
		},
		Attributes: []discovery_kit_api.Attribute{
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.cluster-name",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.distribution",
			}, {
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.namespace",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.container.name",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.container.ready",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.container.image",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.service.name",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.pod.name",
			},
			{
				Matcher: discovery_kit_api.StartsWith,
				Name:    "k8s.pod.label.",
			},
			{
				Matcher: discovery_kit_api.StartsWith,
				Name:    "k8s.namespace.label.",
			},
			{
				Matcher: discovery_kit_api.StartsWith,
				Name:    "k8s.deployment.label.",
			},
			{
				Matcher: discovery_kit_api.StartsWith,
				Name:    "k8s.label.",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.replicaset",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.daemonset",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.deployment",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.workload-type",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.workload-owner",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "k8s.statefulset",
			},
		},
	}
}

func getContainerToJvmEnrichmentRule() discovery_kit_api.TargetEnrichmentRule {
	return discovery_kit_api.TargetEnrichmentRule{
		Id:      "com.steadybit.extension_jvm.container-to-jvm",
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		Src: discovery_kit_api.SourceOrDestination{
			Type: "com.steadybit.extension_container.container",
			Selector: map[string]string{
				"container.id.stripped": "${dest.container.id.stripped}",
			},
		},
		Dest: discovery_kit_api.SourceOrDestination{
			Type: targetID,
			Selector: map[string]string{
				"container.id.stripped": "${src.container.id.stripped}",
			},
		},
		Attributes: []discovery_kit_api.Attribute{
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "container.host",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "container.name",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "container.id",
			},
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "container.image",
			},
			{
				Matcher: discovery_kit_api.StartsWith,
				Name:    "container.label.",
			},
		},
	}
}

func getJvmToContainerEnrichmentRule() discovery_kit_api.TargetEnrichmentRule {
	return discovery_kit_api.TargetEnrichmentRule{
		Id:      "com.steadybit.extension_jvm.jvm-to-container",
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		Src: discovery_kit_api.SourceOrDestination{
			Type: targetID,
			Selector: map[string]string{
				"container.id.stripped": "${dest.container.id.stripped}",
			},
		},
		Dest: discovery_kit_api.SourceOrDestination{
			Type: "com.steadybit.extension_container.container",
			Selector: map[string]string{
				"container.id.stripped": "${src.container.id.stripped}",
			},
		},
		Attributes: []discovery_kit_api.Attribute{
			{
				Matcher: discovery_kit_api.Equals,
				Name:    "jvm-instance.name",
			},
		},
	}
}

func (j *jvmDiscovery) DescribeAttributes() []discovery_kit_api.AttributeDescription {
	return []discovery_kit_api.AttributeDescription{
		{
			Attribute: "jvm-instance.name",
			Label: discovery_kit_api.PluralLabel{
				One:   "Instance Name",
				Other: "Instance Names",
			},
		},
		{
			Attribute: "instance.type",
			Label: discovery_kit_api.PluralLabel{
				One:   "Instance Type",
				Other: "Instance Types",
			},
		}, {
			Attribute: "instance.hostname",
			Label: discovery_kit_api.PluralLabel{
				One:   "Instance Hostname",
				Other: "Instance Hostnames",
			},
		},
		{
			Attribute: "process.pid",
			Label: discovery_kit_api.PluralLabel{
				One:   "Process Pid",
				Other: "Process Pids",
			},
		},
		{
			Attribute: "k8s.container.name",
			Label: discovery_kit_api.PluralLabel{
				One:   "container name",
				Other: "container names",
			},
		},
		{
			Attribute: "k8s.namespace",
			Label: discovery_kit_api.PluralLabel{
				One:   "Namespace name",
				Other: "Namespace names",
			},
		},
		{
			Attribute: "k8s.cluster-name",
			Label: discovery_kit_api.PluralLabel{
				One:   "Cluster name",
				Other: "Cluster names",
			},
		},
		{
			Attribute: "k8s.deployment",
			Label: discovery_kit_api.PluralLabel{
				One:   "deployment name",
				Other: "deployment names",
			},
		},
	}
}

func (j *jvmDiscovery) DiscoverTargets(_ context.Context) ([]discovery_kit_api.Target, error) {
	javaVms := j.jvms.GetJvms()
	targets := make([]discovery_kit_api.Target, 0, len(javaVms))

	for _, javaVm := range javaVms {
		target := discovery_kit_api.Target{
			Id:         fmt.Sprintf("%s/%d", javaVm.Hostname(), javaVm.Pid()),
			TargetType: targetID,
			Label:      "?",
			Attributes: map[string][]string{
				"instance.type":     {"java"},
				"process.pid":       {strconv.Itoa(int(javaVm.Pid()))},
				"instance.hostname": {javaVm.Hostname()},
				"host.hostname":     {javaVm.Hostname()},
				"host.domainname":   {javaVm.HostFQDN()},
			},
		}
		if name := getApplicationName(javaVm); name != "" {
			target.Label = name
			target.Attributes["jvm-instance.name"] = []string{name}
		}
		if c, ok := javaVm.(jvm.JavaVmInContainer); ok {
			target.Attributes["container.id.stripped"] = []string{c.ContainerId()}
		}
		targets = append(targets, target)
	}

	j.enhanceTargetsWithSpringAttributes(targets)
	j.enhanceTargetsWithDataSourceAttributes(targets)
	return discovery_kit_commons.ApplyAttributeExcludes(targets, config.Config.DiscoveryAttributesExcludesJVM), nil
}

func (j *jvmDiscovery) enhanceTargetsWithDataSourceAttributes(targets []discovery_kit_api.Target) {
	for _, app := range j.datasource.getApplications() {
		target := findTargetByPid(targets, app.Pid)
		if target != nil {
			for _, dataSourceConnection := range app.DataSourceConnections {
				target.Attributes["datasource.jdbc-url"] = append(target.Attributes["datasource.jdbc-url"], dataSourceConnection.JdbcUrl)
			}
		}
	}
}

func (j *jvmDiscovery) enhanceTargetsWithSpringAttributes(targets []discovery_kit_api.Target) {
	for _, app := range j.spring.getApplications() {
		target := findTargetByPid(targets, app.Pid)
		if target != nil {
			target.Attributes["jvm-instance.name"] = utils.AppendIfMissing(target.Attributes["jvm-instance.name"], app.Name)
			target.Attributes["spring-instance.name"] = []string{app.Name}
			target.Attributes["instance.type"] = append(target.Attributes["instance.type"], "spring")
			if app.SpringBoot {
				target.Attributes["instance.type"] = append(target.Attributes["instance.type"], "spring-boot")
			}
			if app.UsingJdbcTemplate {
				target.Attributes["spring-instance.jdbc-template"] = []string{"true"}
			}
			if app.UsingHttpClient {
				target.Attributes["spring-instance.http-client"] = []string{"true"}
			}
			addMvcMappings(target, app.MvcMappings)
			addHttpClientRequests(target, app.HttpClientRequests)
		}
	}
}

func addHttpClientRequests(target *discovery_kit_api.Target, requests []HttpRequest) {
	if len(requests) == 0 {
		return
	}
	for _, request := range requests {
		target.Attributes["spring-instance.http-outgoing-calls"] = append(target.Attributes["spring-instance.http-outgoing-calls"], request.Address)
		if !request.CircuitBreaker {
			target.Attributes["spring-instance.http-outgoing-calls.missing-circuit-breaker"] = append(target.Attributes["spring-instance.http-outgoing-calls"], request.Address)
		}
		if request.Timeout == 0 {
			target.Attributes["spring-instance.http-outgoing-calls.missing-timeout"] = append(target.Attributes["spring-instance.http-outgoing-calls"], request.Address)
		}
	}
}

func addMvcMappings(target *discovery_kit_api.Target, mappings []SpringMvcMapping) {
	if len(mappings) == 0 {
		return
	}
	mappingsByPath := make(map[string]SpringMvcMapping)
	for _, mapping := range mappings {
		if mapping.Patterns != nil {
			for _, pattern := range mapping.Patterns {
				mappingsByPath[pattern] = mapping
			}
		}
	}
	log.Trace().Msgf("mappingsByPath: %v", mappingsByPath)
	for pattern := range mappingsByPath {
		target.Attributes["spring-instance.mvc-mapping"] = append(target.Attributes["spring-instance.mvc-mapping"], pattern)
	}
}

func findTargetByPid(targets []discovery_kit_api.Target, pid int32) *discovery_kit_api.Target {
	for _, target := range targets {
		if target.Attributes["process.pid"][0] == strconv.Itoa(int(pid)) {
			return extutil.Ptr(target)
		}
	}
	return nil
}

func getApplicationName(jvm jvm.JavaVm) string {
	name := strings.Replace(jvm.MainClass(), ".jar", "", -1)
	return strings.TrimPrefix(name, "/")
}
