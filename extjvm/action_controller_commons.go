package extjvm

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"slices"
)

var (
	patternAttribute = action_kit_api.ActionParameter{
		Name:        "pattern",
		Label:       "Request Mapping",
		Description: extutil.Ptr("Which request mapping pattern should be used to match the requests?"),
		Type:        action_kit_api.String,
		Required:    extutil.Ptr(true),
		Options: extutil.Ptr([]action_kit_api.ParameterOption{
			action_kit_api.ParameterOptionsFromTargetAttribute{
				Attribute: "spring-instance.mvc-mapping",
			},
		}),
	}
	methodAttribute = action_kit_api.ActionParameter{
		Name:               "method",
		Label:              "Http Method",
		Description:        extutil.Ptr("Which HTTP method should be attacked?"),
		Type:               action_kit_api.String,
		Options:            methodsOptions,
		Deprecated:         extutil.Ptr(true),
		DeprecationMessage: extutil.Ptr("Use the 'Http Methods' parameter instead."),
	}
	methodsAttribute = action_kit_api.ActionParameter{
		Name:         "methods",
		Label:        "Http Methods",
		Description:  extutil.Ptr("Which HTTP methods should be attacked?"),
		Type:         action_kit_api.StringArray,
		Required:     extutil.Ptr(true),
		DefaultValue: extutil.Ptr("*"),
		Options:      methodsOptions,
	}
	methodsOptions = extutil.Ptr([]action_kit_api.ParameterOption{
		action_kit_api.ExplicitParameterOption{
			Label: "Any",
			Value: "*",
		},
		action_kit_api.ExplicitParameterOption{
			Label: "GET",
			Value: "GET",
		},
		action_kit_api.ExplicitParameterOption{
			Label: "HEAD",
			Value: "HEAD",
		},
		action_kit_api.ExplicitParameterOption{
			Label: "OPTIONS",
			Value: "OPTIONS",
		},
		action_kit_api.ExplicitParameterOption{
			Label: "TRACE",
			Value: "TRACE",
		},
		action_kit_api.ExplicitParameterOption{
			Label: "POST",
			Value: "POST",
		},
		action_kit_api.ExplicitParameterOption{
			Label: "PUT",
			Value: "PUT",
		},
		action_kit_api.ExplicitParameterOption{
			Label: "PATCH",
			Value: "PATCH",
		},
		action_kit_api.ExplicitParameterOption{
			Label: "DELETE",
			Value: "DELETE",
		},
	})
)

func extractPattern(request action_kit_api.PrepareActionRequestBody) (string, error) {
	pattern := extutil.ToString(request.Config["pattern"])
	if pattern == "" {
		return "", errors.New("pattern is required")
	}
	return pattern, nil
}

func extractMethods(request action_kit_api.PrepareActionRequestBody) ([]string, error) {
	httpMethods := make([]string, 0)

	method := extutil.ToString(request.Config["method"])
	if method != "" {
		log.Info().Msg("`HTTP Method` is deprecated, use `HTTP Methods` instead.")
		httpMethods = append(httpMethods, method)
	}

	httpMethods = append(httpMethods, extutil.ToStringArray(request.Config["methods"])...)
	if len(httpMethods) == 0 {
		return httpMethods, errors.New("pattern is required")
	}

	return httpMethods, nil
}

func extractHandlerMethods(spring *SpringDiscovery, request action_kit_api.PrepareActionRequestBody) ([]string, error) {
	pattern, err := extractPattern(request)
	if err != nil {
		return nil, err
	}

	methods, err := extractMethods(request)
	if err != nil {
		return nil, err
	}

	pid, err := extractPid(request)
	if err != nil {
		return nil, err
	}

	application := spring.findApplication(pid)
	if application == nil {
		return nil, errors.New("spring instance not found")
	}

	if application.MvcMappings == nil {
		return nil, errors.New("spring MVC mappings not found")
	}

	relevantMappings := make([]SpringMvcMapping, 0)
	for _, mapping := range application.MvcMappings {
		if !slices.Contains(mapping.Patterns, pattern) {
			continue
		}
		if slices.Contains(methods, "*") || (len(mapping.Methods) == 0 && slices.Contains(methods, "GET")) {
			relevantMappings = append(relevantMappings, mapping)
		} else {
			for _, mappingMethod := range mapping.Methods {
				if slices.Contains(methods, mappingMethod) {
					relevantMappings = append(relevantMappings, mapping)
					break
				}
			}
		}
	}

	configMethods := make([]string, 0)
	for _, m := range relevantMappings {
		configMethods = append(configMethods, fmt.Sprintf("%s#%s", m.HandlerClass, m.HandlerName))
	}
	return configMethods, nil
}
