/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extjvm

import (
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"time"
)

func NewControllerException() action_kit_sdk.Action[JavaagentActionState] {
	return &javaagentAction{
		pluginJar:      utils.GetJarPath("attack-java-javaagent.jar"),
		description:    controllerExceptionDescribe(),
		configProvider: controllerExceptionConfigProvider,
	}
}

func controllerExceptionDescribe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          ActionIDPrefix + ".spring-mvc-exception-attack",
		Label:       "Controller Exception",
		Description: "Throw an exception in an Spring MVC controller method",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(controllerExceptionIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			// The target type this action is for
			TargetType: targetID + "(instance.type=spring;spring-instance.mvc-mapping)",
			// You can provide a list of target templates to help the user select targets.
			// A template can be used to pre-fill a selection
			SelectionTemplates: extutil.Ptr(targetSelectionTemplates),
		}),
		// Category for the targets to appear in
		Category: extutil.Ptr(category),

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
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  extutil.Ptr("How long should the traffic be dropped?"),
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
			},
			erroneousCallRate,
		},
		Stop: extutil.Ptr(action_kit_api.MutatingEndpointReference{}),
	}
}

func controllerExceptionConfigProvider(request action_kit_api.PrepareActionRequestBody) (any, error) {
	duration, err := extractDuration(request)
	if err != nil {
		return nil, err
	}

	handlerMethods, err := extractHandlerMethods(request)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"attack-class":      "com.steadybit.attacks.javaagent.instrumentation.JavaMethodExceptionInstrumentation",
		"duration":          int(duration / time.Millisecond),
		"erroneousCallRate": extutil.ToInt(request.Config["erroneousCallRate"]),
		"methods":           handlerMethods,
	}, nil
}
