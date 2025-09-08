/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extjvm

import (
	"fmt"
	"time"

	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
)

func NewJavaMethodException(facade jvm.JavaFacade) action_kit_sdk.Action[JavaagentActionState] {
	return &javaagentAction{
		pluginJar:      "attack-java-javaagent.jar",
		description:    methodExceptionDescribe(),
		configProvider: methodExceptionConfigProvider,
		facade:         facade,
	}
}

func methodExceptionDescribe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          ActionIDPrefix + ".java-method-exception-attack",
		Label:       "Java Method Exception",
		Description: "Throw an exception in an public Java method.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(javaMethodExceptionIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			// The target type this action is for
			TargetType: targetType + "(instance.type=java)",
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
				Name:        "className",
				Label:       "Class Name",
				Description: extutil.Ptr("Which Java class should be attacked?"),
				Type:        action_kit_api.ActionParameterTypeString,
				Required:    extutil.Ptr(true),
			},
			{
				Name:        "methodName",
				Label:       "Method Name",
				Description: extutil.Ptr("Which public method should be attacked?"),
				Type:        action_kit_api.ActionParameterTypeString,
				Required:    extutil.Ptr(true),
			},
			erroneousCallRate,
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  extutil.Ptr("How long should the delay be inflicted?"),
				Type:         action_kit_api.ActionParameterTypeDuration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
			},
			{
				Name:         "validate",
				Label:        "Validate class and method name",
				Description:  extutil.Ptr("Should the action fail if the specified class and method could not be found?"),
				Type:         action_kit_api.ActionParameterTypeBoolean,
				DefaultValue: extutil.Ptr("true"),
				Required:     extutil.Ptr(true),
				Advanced:     extutil.Ptr(true),
			},
		},
		Stop: extutil.Ptr(action_kit_api.MutatingEndpointReference{}),
	}
}

func methodExceptionConfigProvider(request action_kit_api.PrepareActionRequestBody) (map[string]interface{}, error) {
	duration, err := extractDuration(request)
	if err != nil {
		return nil, err
	}

	className := extutil.ToString(request.Config["className"])
	methodName := extutil.ToString(request.Config["methodName"])

	return map[string]interface{}{
		"attack-class":      "com.steadybit.attacks.javaagent.instrumentation.JavaMethodExceptionInstrumentation",
		"duration":          int(duration / time.Millisecond),
		"erroneousCallRate": extutil.ToInt(request.Config["erroneousCallRate"]),
		"methods":           []string{fmt.Sprintf("%s#%s", className, methodName)},
	}, nil
}
