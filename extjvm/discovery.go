/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extjvm

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/discovery-kit/go/discovery_kit_commons"
	"github.com/steadybit/discovery-kit/go/discovery_kit_sdk"
	"github.com/steadybit/extension-jvm/config"
	"github.com/steadybit/extension-jvm/extjvm/controller"
	"github.com/steadybit/extension-jvm/extjvm/hotspot"
	"github.com/steadybit/extension-jvm/extjvm/java_process"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"strconv"
	"strings"
	"time"
)

type jvmDiscovery struct {
}

var (
	_ discovery_kit_sdk.TargetDescriber          = (*jvmDiscovery)(nil)
	_ discovery_kit_sdk.AttributeDescriber       = (*jvmDiscovery)(nil)
	_ discovery_kit_sdk.EnrichmentRulesDescriber = (*jvmDiscovery)(nil)
)

func NewJvmDiscovery() discovery_kit_sdk.TargetDiscovery {
	discovery := &jvmDiscovery{}
	return discovery_kit_sdk.NewCachedTargetDiscovery(discovery,
		discovery_kit_sdk.WithRefreshTargetsNow(),
		discovery_kit_sdk.WithRefreshTargetsInterval(context.Background(), 30*time.Second),
	)
}

func StartJvmInfrastructure() {
	installSignalHandler()

	controller.Start(config.Config.JavaAgentAttachmentPort)

	initDataSourceDiscovery()
	initSpringDiscovery()
	addJVMListener()

	startAttachment()

	// Start JVM Watcher
	java_process.Start()
	// Start Hotspot JVM Watcher
	hotspot.Start()
}

func (j *jvmDiscovery) Describe() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         targetID,
		RestrictTo: extutil.Ptr(discovery_kit_api.LEADER),
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
	vms := GetJVMs()
	targets := make([]discovery_kit_api.Target, 0, len(vms))
	for _, vm := range vms {
		targets = append(targets, discovery_kit_api.Target{
			Id:         fmt.Sprintf("%s/%d", vm.Hostname, vm.Pid),
			TargetType: targetID,
			Label:      getApplicationName(vm, "?"),
			Attributes: map[string][]string{
				"instance.type":         {"java"},
				"jvm-instance.name":     {getApplicationName(vm, "")},
				"container.id.stripped": {vm.ContainerId},
				"process.pid":           {strconv.Itoa(int(vm.Pid))},
				"instance.hostname":     {vm.Hostname},
				"host.hostname":         {vm.Hostname},
			},
		})
	}
	enhanceTargetsWithSpringAttributes(targets)
	enhanceTargetsWithDataSourceAttributes(targets)
	return discovery_kit_commons.ApplyAttributeExcludes(targets, config.Config.DiscoveryAttributesExcludesJVM), nil
}

func enhanceTargetsWithDataSourceAttributes(targets []discovery_kit_api.Target) {
	dataSourceApplications := GetDataSourceApplications()
	for _, dataSourceApplication := range dataSourceApplications {
		target := findTargetByPid(targets, dataSourceApplication.Pid)
		if target != nil {
			for _, dataSourceConnection := range dataSourceApplication.DataSourceConnections {
				target.Attributes["datasource.jdbc-url"] = append(target.Attributes["datasource.jdbc-url"], dataSourceConnection.JdbcUrl)
			}
		}
	}
}

func enhanceTargetsWithSpringAttributes(targets []discovery_kit_api.Target) {
	springApplications := GetSpringApplications()
	for _, app := range springApplications {
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

func addHttpClientRequests(target *discovery_kit_api.Target, requests *[]HttpRequest) {
	if requests == nil {
		return
	}
	for _, request := range *requests {
		target.Attributes["spring-instance.http-outgoing-calls"] = append(target.Attributes["spring-instance.http-outgoing-calls"], request.Address)
		if !request.CircuitBreaker {
			target.Attributes["spring-instance.http-outgoing-calls.missing-circuit-breaker"] = append(target.Attributes["spring-instance.http-outgoing-calls"], request.Address)
		}
		if request.Timeout == 0 {
			target.Attributes["spring-instance.http-outgoing-calls.missing-timeout"] = append(target.Attributes["spring-instance.http-outgoing-calls"], request.Address)
		}
	}
}

func addMvcMappings(target *discovery_kit_api.Target, mappings *[]SpringMvcMapping) {
	if mappings == nil {
		return
	}
	mappingsByPath := make(map[string]SpringMvcMapping)
	for _, mapping := range *mappings {
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

func getApplicationName(jvm jvm.JavaVm, defaultIfEmpty string) string {
	name := strings.Replace(jvm.MainClass, ".jar", "", -1)
	if name == "" {
		name = defaultIfEmpty
	}
	name = strings.TrimPrefix(name, "/")
	return name
}
