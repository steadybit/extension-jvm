/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extjvm

import (
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/exthttp"
	"github.com/steadybit/extension-kit/extutil"
	"net/http"
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
	targets := make([]discovery_kit_api.Target, 0)
	//for i, name := range config.Config.RobotNames {
	//	targets[i] = discovery_kit_api.Target{
	//		Id:         name,
	//		TargetType: targetID,
	//		Label:      name,
	//		Attributes: map[string][]string{"robot.reportedBy": {"extension-jvm"}},
	//	}
	//}
	exthttp.WriteBody(w, discovery_kit_api.DiscoveredTargets{Targets: targets})
}
