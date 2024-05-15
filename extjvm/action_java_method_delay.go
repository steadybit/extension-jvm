/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extjvm

import (
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"time"
)

func NewJavaMethodDelay() action_kit_sdk.Action[JavaagentActionState] {
	return &javaagentAction{
		pluginJar:      utils.GetJarPath("attack-java-javaagent.jar"),
		description:    methodDelayDescribe(),
		configProvider: methodDelayConfigProvider,
	}
}

func methodDelayDescribe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          ActionIDPrefix + ".java-method-delay-attack",
		Label:       "Java Method Delay",
		Description: "Delay a public method call by the given duration.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(javaMethodDelayIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			// The target type this action is for
			TargetType: targetID + "(instance.type=java)",
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
			{
				Name:        "className",
				Label:       "Class Name",
				Description: extutil.Ptr("Which Java class should be attacked?"),
				Type:        action_kit_api.String,
				Required:    extutil.Ptr(true),
			},
			{
				Name:        "methodName",
				Label:       "Method Name",
				Description: extutil.Ptr("Which public method should be attacked?"),
				Type:        action_kit_api.String,
				Required:    extutil.Ptr(true),
			},
			{
				Name:         "delay",
				Label:        "Delay",
				Description:  extutil.Ptr("How long should the db access be delayed?"),
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("500ms"),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  extutil.Ptr("How long should the delay be inflicted?"),
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "delayJitter",
				Label:        "Jitter",
				Description:  extutil.Ptr("Add random +/-30% jitter to response delay?"),
				Type:         action_kit_api.Boolean,
				DefaultValue: extutil.Ptr("true"),
				Required:     extutil.Ptr(true),
				Advanced:     extutil.Ptr(true),
			},
		},
		Stop: extutil.Ptr(action_kit_api.MutatingEndpointReference{}),
	}
}

func methodDelayConfigProvider(request action_kit_api.PrepareActionRequestBody) (any, error) {
	duration, err := extractDuration(request)
	if err != nil {
		return nil, err
	}

	className := extutil.ToString(request.Config["className"])
	methodName := extutil.ToString(request.Config["methodName"])

	return map[string]interface{}{
		"attack-class": "com.steadybit.attacks.javaagent.instrumentation.JavaMethodDelayInstrumentation",
		"duration":     int(duration / time.Millisecond),
		"delay":        extutil.ToUInt64(request.Config["delay"]),
		"delayJitter":  extutil.ToBool(request.Config["delayJitter"]),
		"methods":      []string{fmt.Sprintf("%s#%s", className, methodName)},
	}, nil
}
