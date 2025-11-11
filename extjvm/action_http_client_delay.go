/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extjvm

import (
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
)

func NewHttpClientDelay(facade jvm.JavaFacade) action_kit_sdk.Action[JavaagentActionState] {
	return &javaagentAction{
		pluginJar:      "attack-java-javaagent.jar",
		description:    httpClientDelayDescribe(),
		configProvider: httpClientDelayConfigProvider,
		facade:         facade,
	}
}

func httpClientDelayDescribe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          ActionIDPrefix + ".spring-httpclient-delay-attack",
		Label:       "Http Client Delay",
		Description: "Delays a response from a RestTemplate or WebClient by the given duration.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(springHttpDelayIcon),
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
				Description:  extutil.Ptr("How long should the delay be inflicted?"),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "delay",
				Label:        "Delay",
				Description:  extutil.Ptr("How long should the db access be delayed?"),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("500ms"),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "delayJitter",
				Label:        "Jitter",
				Description:  extutil.Ptr("Add random +/-30% jitter to response delay?"),
				Type:         action_kit_api.ActionParameterTypeBoolean,
				DefaultValue: extutil.Ptr("false"),
				Required:     extutil.Ptr(true),
				Advanced:     extutil.Ptr(true),
			},
			{
				Name:        "httpMethods",
				Label:       "Http Methods",
				Description: extutil.Ptr("Which HTTP methods should be attacked?"),
				Type:        action_kit_api.ActionParameterTypeStringArray,
				Required:    extutil.Ptr(false),
				Advanced:    extutil.Ptr(true),
				Options:     methodsOptions,
			},
			{
				Name:         "hostAddress",
				Label:        "Host Address",
				Description:  extutil.Ptr("Request to which host address should be attacked?"),
				Type:         action_kit_api.ActionParameterTypeString,
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
				Description:  extutil.Ptr("Which URL path should be attacked? Use '*' or empty for any. All paths starting with the given value will be matched."),
				Type:         action_kit_api.ActionParameterTypeString,
				DefaultValue: extutil.Ptr(""),
				Required:     extutil.Ptr(false),
				Advanced:     extutil.Ptr(true),
			},
		},
		Stop: extutil.Ptr(action_kit_api.MutatingEndpointReference{}),
	}

}

func httpClientDelayConfigProvider(request action_kit_api.PrepareActionRequestBody) (map[string]interface{}, error) {
	duration, err := extractDuration(request)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"attack-class": "com.steadybit.attacks.javaagent.instrumentation.SpringHttpClientDelayInstrumentation",
		"duration":     int(duration / time.Millisecond),
		"delay":        extutil.ToUInt64(request.Config["delay"]),
		"delayJitter":  extutil.ToBool(request.Config["delayJitter"]),
		"httpMethods":  extutil.ToStringArray(request.Config["httpMethods"]),
		"hostAddress":  extutil.ToString(request.Config["hostAddress"]),
		"urlPath":      extutil.ToString(request.Config["urlPath"]),
	}, nil
}
