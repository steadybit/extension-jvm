/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extjvm

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"time"
)

type controllerException struct{}

type ControllerExceptionState struct {
	ErroneousCallRate int
	*ControllerState
}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[ControllerExceptionState]         = (*controllerException)(nil)
	_ action_kit_sdk.ActionWithStop[ControllerExceptionState] = (*controllerException)(nil) // Optional, needed when the action needs a stop method

)

func NewControllerException() action_kit_sdk.Action[ControllerExceptionState] {
	return &controllerException{}
}

func (l *controllerException) NewEmptyState() ControllerExceptionState {
	return ControllerExceptionState{
		ControllerState: &ControllerState{
			AttackState: &AttackState{},
		},
	}
}

// Describe returns the action description for the platform with all required information.
func (l *controllerException) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          TargetIDPrefix + ".spring-mvc-exception-attack",
		Label:       "Controller Exception",
		Description: "Throw an exception in an Spring MVC controller method",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(controllerExceptionIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			// The target type this action is for
			TargetType: targetID + "(application.type=spring;spring.mvc-mapping)",
			// You can provide a list of target templates to help the user select targets.
			// A template can be used to pre-fill a selection
			SelectionTemplates: extutil.Ptr(targetSelectionTemplates),
		}),
		// Category for the targets to appear in
		Category: extutil.Ptr("JVM Application Attacks"),

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
		TimeControl: action_kit_api.External,

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

// Prepare is called before the action is started.
// It can be used to validate the parameters and prepare the action.
// It must not cause any harmful effects.
// The passed in state is included in the subsequent calls to start/status/stop.
// So the state should contain all information needed to execute the action and even more important: to be able to stop it.
func (l *controllerException) Prepare(_ context.Context, state *ControllerExceptionState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	errResult := extractPattern(request, state.ControllerState)
	if errResult != nil {
		return errResult, nil
	}

	errResult = extractMethod(request, state.ControllerState)
	if errResult != nil {
		return errResult, nil
	}

	erroneousCallRate := extutil.ToInt(request.Config["erroneousCallRate"])
	state.ErroneousCallRate = erroneousCallRate

	errResult = extractDuration(request, state.AttackState)
	if errResult != nil {
		return errResult, nil
	}

	errResult = extractPid(request, state.AttackState)
	if errResult != nil {
		return errResult, nil
	}

	errResult = extractHandlerMethods(request, state.ControllerState)
	if errResult != nil {
		return errResult, nil
	}

	var config = map[string]interface{}{
		"attack-class":      "com.steadybit.attacks.javaagent.instrumentation.JavaMethodExceptionInstrumentation",
		"duration":          int(state.Duration / time.Millisecond),
		"erroneousCallRate": state.ErroneousCallRate,
		"methods":           state.HandlerMethods,
	}
	return commonPrepareEnd(config, state.AttackState, request)
}

// Start is called to start the action
// You can mutate the state here.
// You can use the result to return messages/errors/metrics or artifacts
func (l *controllerException) Start(_ context.Context, state *ControllerExceptionState) (*action_kit_api.StartResult, error) {
	return commonStart(state.AttackState)
}

// Stop is called to stop the action
// It will be called even if the start method did not complete successfully.
// It should be implemented in a immutable way, as the agent might to retries if the stop method timeouts.
// You can use the result to return messages/errors/metrics or artifacts
func (l *controllerException) Stop(_ context.Context, state *ControllerExceptionState) (*action_kit_api.StopResult, error) {
	return commonStop(state.AttackState)
}
