package extjvm

import (
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"github.com/steadybit/extension-kit/extutil"
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

type ControllerState struct {
	Pattern        string
	HttpMethods    []string
	HandlerMethods []string
	*AttackState
}

func extractPattern(request action_kit_api.PrepareActionRequestBody, state *ControllerState) *action_kit_api.PrepareResult {
	pattern := extutil.ToString(request.Config["pattern"])
	if pattern == "" {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Pattern is required",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}
	}
	state.Pattern = pattern
	return nil
}

func extractHttpMethods(request action_kit_api.PrepareActionRequestBody, state *ControllerState) *action_kit_api.PrepareResult {
	state.HttpMethods = make([]string, 0)

	method := extutil.ToString(request.Config["method"])
	if method != "" {
		log.Info().Msg("`HTTP Method` is deprecated, use `HTTP Methods` instead.")
		state.HttpMethods = append(state.HttpMethods, method)
	}

	state.HttpMethods = append(state.HttpMethods, extutil.ToStringArray(request.Config["methods"])...)
	if len(state.HttpMethods) == 0 {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Method is required",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}
	}
	return nil
}

func extractHandlerMethods(_ action_kit_api.PrepareActionRequestBody, state *ControllerState) *action_kit_api.PrepareResult {
	application := FindSpringApplication(state.Pid)
	if application == nil {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Spring instance not found",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}
	}

	if application.MvcMappings == nil {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Spring MVC mappings not found",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}
	}
	relevantMappings := make([]SpringMvcMapping, 0)
	for _, m := range *application.MvcMappings {
		if utils.ContainsString(m.Patterns, state.Pattern) {
			if utils.ContainsString(state.HttpMethods, "*") || (len(m.Methods) == 0 && utils.ContainsString(state.HttpMethods, "GET")) {
				relevantMappings = append(relevantMappings, m)
			} else {
				for _, method := range m.Methods {
					if utils.ContainsString(state.HttpMethods, method) {
						relevantMappings = append(relevantMappings, m)
						break
					}
				}
			}
		}
	}
	configMethods := make([]string, 0)
	for _, m := range relevantMappings {
		method := fmt.Sprintf("%s#%s", m.HandlerClass, m.HandlerName)
		configMethods = append(configMethods, method)
	}
	state.HandlerMethods = configMethods
	return nil
}
