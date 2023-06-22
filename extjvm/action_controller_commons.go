package extjvm

import (
	"fmt"
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
				Attribute: "spring.mvc-mapping",
			},
		}),
	}
	methodAttribute = action_kit_api.ActionParameter{
		Name:         "method",
		Label:        "Http Method",
		Description:  extutil.Ptr("Which HTTP methods should be attacked?"),
		Type:         action_kit_api.String,
		Required:     extutil.Ptr(true),
		DefaultValue: extutil.Ptr("*"),
		Options: methodsOptions,
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
	Method         string
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

func extractMethod(request action_kit_api.PrepareActionRequestBody, state *ControllerState) *action_kit_api.PrepareResult {
	method := extutil.ToString(request.Config["method"])
	if method == "" {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Method is required",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}
	}
	state.Method = method
	return nil
}

func extractHandlerMethods(_ action_kit_api.PrepareActionRequestBody, state *ControllerState) *action_kit_api.PrepareResult {
	application := FindSpringApplication(state.Pid)
	if application == nil {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Spring application not found",
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
			if state.Method == "*" || (len(m.Methods) == 0 && state.Method == "GET") {
				relevantMappings = append(relevantMappings, m)
			} else if utils.ContainsString(m.Methods, state.Method) {
				relevantMappings = append(relevantMappings, m)
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
