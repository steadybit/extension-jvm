/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extjvm

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"net/http"
	"strconv"
	"strings"
)

const discoveryBasePath = basePath + "/discovery"

func RegisterDiscoveryHandlers() {
	exthttp.RegisterHttpHandler(discoveryBasePath, exthttp.GetterAsHandler(getDiscoveryDescription))
	exthttp.RegisterHttpHandler(discoveryBasePath+"/target-description", exthttp.GetterAsHandler(getTargetDescription))
	exthttp.RegisterHttpHandler(discoveryBasePath+"/attribute-descriptions", exthttp.GetterAsHandler(getAttributeDescriptions))
	exthttp.RegisterHttpHandler(discoveryBasePath+"/discovered-targets", getDiscoveredTargets)
}

func GetDiscoveryList() discovery_kit_api.DiscoveryList {
	return discovery_kit_api.DiscoveryList{
		Discoveries: []discovery_kit_api.DescribingEndpointReference{
			{
				Method: "GET",
				Path:   discoveryBasePath,
			},
		},
		TargetTypes: []discovery_kit_api.DescribingEndpointReference{
			{
				Method: "GET",
				Path:   discoveryBasePath + "/target-description",
			},
		},
		TargetAttributes: []discovery_kit_api.DescribingEndpointReference{
			{
				Method: "GET",
				Path:   discoveryBasePath + "/attribute-descriptions",
			},
		},
	}
}

func getDiscoveryDescription() discovery_kit_api.DiscoveryDescription {
	return discovery_kit_api.DiscoveryDescription{
		Id:         targetID,
		RestrictTo: extutil.Ptr(discovery_kit_api.LEADER),
		Discover: discovery_kit_api.DescribingEndpointReferenceWithCallInterval{
			Method:       "GET",
			Path:         discoveryBasePath + "/discovered-targets",
			CallInterval: extutil.Ptr("1m"),
		},
	}
}

func getTargetDescription() discovery_kit_api.TargetDescription {
	return discovery_kit_api.TargetDescription{
		Id:      targetID,
		Version: extbuild.GetSemverVersionStringOrUnknown(),
		Icon:    extutil.Ptr(targetIcon),

		// Labels used in the UI
		Label: discovery_kit_api.PluralLabel{One: "JVM application", Other: "JVM applications"},

		// Category for the targets to appear in
		Category: extutil.Ptr("JVM Application Attacks"),

		// Specify attributes shown in table columns and to be used for sorting
		Table: discovery_kit_api.Table{
			Columns: []discovery_kit_api.Column{
				{Attribute: "application.name"},
				{Attribute: "application.type"},
				{Attribute: "k8s.namespace"},
				{Attribute: "k8s.cluster-name"},
				{Attribute: "k8s.deployment"},
			},
			OrderBy: []discovery_kit_api.OrderBy{
				{
					Attribute: "application.name",
					Direction: "ASC",
				},
			},
		},
	}
}

func getAttributeDescriptions() discovery_kit_api.AttributeDescriptions {
	return discovery_kit_api.AttributeDescriptions{
		Attributes: []discovery_kit_api.AttributeDescription{
			{
				Attribute: "application.name",
				Label: discovery_kit_api.PluralLabel{
					One:   "Application Name",
					Other: "Application Names",
				},
			},
			{
				Attribute: "application.type",
				Label: discovery_kit_api.PluralLabel{
					One:   "Application Type",
					Other: "Application Types",
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
					One:   "namespace name",
					Other: "namespace names",
				},
			},
			{
				Attribute: "k8s.cluster-name",
				Label: discovery_kit_api.PluralLabel{
					One:   "cluster name",
					Other: "cluster names",
				},
			},
			{
				Attribute: "k8s.deployment",
				Label: discovery_kit_api.PluralLabel{
					One:   "deployment name",
					Other: "deployment names",
				},
			},
		},
	}
}

func getDiscoveredTargets(w http.ResponseWriter, _ *http.Request, _ []byte) {
	vms := GetJVMs()
	targets := make([]discovery_kit_api.Target, len(vms))
	for i, jvm := range vms {
		targets[i] = discovery_kit_api.Target{
			Id:         fmt.Sprintf("%d/%s", jvm.Pid, getApplicationName(jvm, "?")),
			TargetType: targetID,
			Label:      getApplicationName(jvm, "?"),
			Attributes: map[string][]string{
				"application.type": {"java"},
				"application.name": {getApplicationName(jvm, "")},
				"container.id":     {jvm.ContainerId},
				"process.pid":      {strconv.Itoa(int(jvm.Pid))},
			},
		}
	}
	// enhance with spring infos
	enhanceTargetsWithSpringAttributes(targets)
	enhanceTargetsWithDataSourceAttributes(targets)
	exthttp.WriteBody(w, discovery_kit_api.DiscoveredTargets{Targets: targets})
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
			target.Attributes["application.name"] = append(target.Attributes["application.name"], app.Name)
			target.Attributes["spring.application.name"] = []string{app.Name}
			target.Attributes["application.type"] = append(target.Attributes["application.type"], "spring")
			if app.SpringBoot {
				target.Attributes["application.type"] = append(target.Attributes["application.type"], "spring-boot")
			}
			if app.UsingJdbcTemplate {
				target.Attributes["spring.jdbc-template"] = []string{"true"}
			}
			if app.UsingHttpClient {
				target.Attributes["spring.http-client"] = []string{"true"}
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
		target.Attributes["application.http-outgoing-calls"] = append(target.Attributes["application.http-outgoing-calls"], request.Address)
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
	for pattern, _ := range mappingsByPath {
		target.Attributes["spring.mvc-mapping"] = append(target.Attributes["spring.mvc-mapping"], pattern)
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
	return name
}
