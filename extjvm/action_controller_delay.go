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

func NewControllerDelay(facade jvm.JavaFacade, spring *SpringDiscovery) action_kit_sdk.Action[JavaagentActionState] {
	return &javaagentAction{
		pluginJar:      "attack-java-javaagent.jar",
		description:    controllerDelayDescribe(),
		configProvider: controllerDelayConfigProvider(spring),
		facade:         facade,
	}
}

func controllerDelayDescribe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          ActionIDPrefix + ".spring-mvc-delay-attack",
		Label:       "Spring Controller Delay",
		Description: "Delay a Spring MVC controller http response by the given duration.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        new(controllerDelayIcon),
		TargetSelection: new(action_kit_api.TargetSelection{
			TargetType:         targetType,
			TargetQuery:        new(`instance.type="spring" AND spring-instance.mvc-mapping IS PRESENT`),
			SelectionTemplates: new(targetSelectionTemplates),
		}),
		Technology: new("JVM"),

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
			patternAttribute,
			methodAttribute,
			methodsAttribute,
			{
				Name:         "delay",
				Label:        "Delay",
				Description:  new("How much should the response be delayed?"),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: new("500ms"),
				Required:     new(true),
			},
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  new("How long should the traffic be dropped?"),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: new("30s"),
				Required:     new(true),
			}, {
				Name:         "delayJitter",
				Label:        "Jitter",
				Description:  new("Add random +/-30% jitter to response delay?"),
				Type:         action_kit_api.ActionParameterTypeBoolean,
				DefaultValue: new("false"),
				Required:     new(true),
				Advanced:     new(true),
			},
		},
		Stop: new(action_kit_api.MutatingEndpointReference{}),
	}
}

func controllerDelayConfigProvider(s *SpringDiscovery) func(request action_kit_api.PrepareActionRequestBody) (map[string]any, error) {
	return func(request action_kit_api.PrepareActionRequestBody) (map[string]any, error) {
		duration, err := extractDuration(request)
		if err != nil {
			return nil, err
		}

		handlerMethods, err := extractHandlerMethods(s, request)
		if err != nil {
			return nil, err
		}

		return map[string]any{
			"attack-class": "com.steadybit.attacks.javaagent.instrumentation.JavaMethodDelayInstrumentation",
			"duration":     int(duration / time.Millisecond),
			"delay":        extutil.ToUInt64(request.Config["delay"]),
			"delayJitter":  extutil.ToBool(request.Config["delayJitter"]),
			"methods":      handlerMethods,
		}, nil
	}
}
