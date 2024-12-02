/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extjvm

import (
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"time"
)

func NewHttpClientStatus(facade jvm.JavaFacade) action_kit_sdk.Action[JavaagentActionState] {
	return &javaagentAction{
		pluginJar:      utils.GetJarPath("attack-java-javaagent.jar"),
		description:    httpClientStatusDescribe(),
		configProvider: httpClientStatusConfigProvider,
		facade:         facade,
	}
}

func httpClientStatusDescribe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          ActionIDPrefix + ".spring-httpclient-status-attack",
		Label:       "Http Client Status",
		Description: "Returns the given status code for a RestTemplate or WebClient call. The original call is not executed.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(springHttpStatusIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			// The target type this action is for
			TargetType: targetType + "(instance.type=spring)",
			// You can provide a list of target templates to help the user select targets.
			// A template can be used to pre-fill a selection
			SelectionTemplates: extutil.Ptr(targetSelectionTemplates),
		}),
		Technology: extutil.Ptr("JVM"),

		// To clarify the purpose of the action, you can set a kind.
		//   Attack: Will cause harm to targets
		//   Check: Will perform checks on the targets
		//   LoadTest: Will perform load tests on the targets
		//   Other
		Kind: action_kit_api.Attack,

		// How the action is controlled over time.
		//   External: The agent takes care and calls stop then the time has passed. Requires a duration parameter. Use this when the duration is known in advance.
		//   Internal: The action has to implement the status endpoint to signal when the action is done. Use this when the duration is not known in advance.
		//   Instantaneous: The action is done immediately. Use this for actions that happen immediately, e.g. a reboot.
		TimeControl: action_kit_api.TimeControlExternal,

		// The parameters for the action
		Parameters: []action_kit_api.ActionParameter{
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  extutil.Ptr("How long should the calls be attacked?"),
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
			},
			{
				Name:        "httpMethods",
				Label:       "Http Methods",
				Description: extutil.Ptr("Which HTTP methods should be attacked?"),
				Type:        action_kit_api.StringArray,
				Required:    extutil.Ptr(false),
				Advanced:    extutil.Ptr(true),
				Options:     methodsOptions,
			},
			{
				Name:         "hostAddress",
				Label:        "Host Address",
				Description:  extutil.Ptr("Request to which host address should be attacked?"),
				Type:         action_kit_api.String,
				DefaultValue: extutil.Ptr("*"),
				Required:     extutil.Ptr(false),
				Advanced:     extutil.Ptr(true),
				OptionsOnly:  extutil.Ptr(false),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "Any",
						Value: "*",
					},
					action_kit_api.ParameterOptionsFromTargetAttribute{
						Attribute: "spring-instance.http-outgoing-calls",
					},
				}),
			},
			{
				Name:         "urlPath",
				Label:        "URL Path",
				Description:  extutil.Ptr("Which URL paths should be attacked? Use '*' or empty for any."),
				Type:         action_kit_api.String,
				DefaultValue: extutil.Ptr(""),
				Required:     extutil.Ptr(false),
				Advanced:     extutil.Ptr(true),
			},
			{
				Name:        "failureCauses",
				Label:       "Failure Types",
				Description: extutil.Ptr("What HTTP client behavior should be simulated? If multiple are selected, one will be chosen randomly for every request."),
				Type:        action_kit_api.StringArray,
				Required:    extutil.Ptr(false),
				Advanced:    extutil.Ptr(true),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "Protocol & Network Errors",
						Value: "ERROR",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Request Timeouts",
						Value: "TIMEOUT",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with 500 status code",
						Value: "HTTP_500",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with 502 status code",
						Value: "HTTP_502",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with 503 status code",
						Value: "HTTP_503",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with 504 status code",
						Value: "HTTP_504",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with a random 5XX status code",
						Value: "HTTP_5XX",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with 400 status code",
						Value: "HTTP_400",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with 403 status code",
						Value: "HTTP_403",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with 404 status code",
						Value: "HTTP_404",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with 429 status code",
						Value: "HTTP_429",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with a random 4XX status code",
						Value: "HTTP_4XX",
					},
				}),
			},
			erroneousCallRate,
		},
		Stop: extutil.Ptr(action_kit_api.MutatingEndpointReference{}),
	}
}

func httpClientStatusConfigProvider(request action_kit_api.PrepareActionRequestBody) (map[string]interface{}, error) {
	duration, err := extractDuration(request)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"attack-class":      "com.steadybit.attacks.javaagent.instrumentation.SpringHttpClientStatusInstrumentation",
		"duration":          int(duration / time.Millisecond),
		"erroneousCallRate": extutil.ToInt(request.Config["erroneousCallRate"]),
		"httpMethods":       extutil.ToStringArray(request.Config["httpMethods"]),
		"hostAddress":       extutil.ToString(request.Config["hostAddress"]),
		"urlPath":           extutil.ToString(request.Config["urlPath"]),
		"failureCauses":     extutil.ToStringArray(request.Config["failureCauses"]),
	}, nil
}
