/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extjvm

import (
	"context"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-jvm/extjvm/attack"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"time"
)

type javaMethodDelay struct{}

type JavaMethodDelayState struct {
	Delay       time.Duration
	DelayJitter bool
	ClassName   string
	MethodName  string
	Validate    bool
	*AttackState
}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[JavaMethodDelayState]         = (*javaMethodDelay)(nil)
	_ action_kit_sdk.ActionWithStop[JavaMethodDelayState] = (*javaMethodDelay)(nil)
)

func NewJavaMethodDelay() action_kit_sdk.Action[JavaMethodDelayState] {
	return &javaMethodDelay{}
}

func (l *javaMethodDelay) NewEmptyState() JavaMethodDelayState {
	return JavaMethodDelayState{
		AttackState: &AttackState{},
	}
}

// Describe returns the action description for the platform with all required information.
func (l *javaMethodDelay) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          ActionIDPrefix + ".java-method-delay-attack",
		Label:       "Java Method Delay",
		Description: "Delay a public method call by the given duration.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(javaMethodDelayIcon),
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
				DefaultValue: extutil.Ptr("false"),
				Required:     extutil.Ptr(true),
				Advanced:     extutil.Ptr(true),
			},
			{
				Name:         "validate",
				Label:        "Validate class and method name",
				Description:  extutil.Ptr("Should the action fail if the specified class and method could not be found?"),
				Type:         action_kit_api.Boolean,
				DefaultValue: extutil.Ptr("true"),
				Required:     extutil.Ptr(true),
				Advanced:     extutil.Ptr(true),
			},
		},
		Stop: extutil.Ptr(action_kit_api.MutatingEndpointReference{}),
	}
}

// Prepare is called before the action is started.
// It can be used to validate the parameters and prepare the action.
// It must not cause any harmful effects.
// The passed in state is included in the subsequent calls to start/status/stop.
// So the state should contain all information needed to execute the action and even more important: to be able to stop it.
func (l *javaMethodDelay) Prepare(_ context.Context, state *JavaMethodDelayState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {

	errResult := extractDuration(request, state.AttackState)
	if errResult != nil {
		return errResult, nil
	}
	parsedDelay := extutil.ToUInt64(request.Config["delay"])
	var delay time.Duration
	if parsedDelay == 0 {
		delay = 0
	} else {
		delay = time.Duration(parsedDelay) * time.Millisecond
	}
	state.Delay = delay

	delayJitter := extutil.ToBool(request.Config["delayJitter"])
	state.DelayJitter = delayJitter
	state.ClassName = extutil.ToString(request.Config["className"])
	state.MethodName = extutil.ToString(request.Config["methodName"])
	state.Validate = extutil.ToBool(request.Config["validate"])

	errResult = extractPid(request, state.AttackState)
	if errResult != nil {
		return errResult, nil
	}

	var config = map[string]interface{}{
		"attack-class": "com.steadybit.attacks.javaagent.instrumentation.JavaMethodDelayInstrumentation",
		"duration":     int(state.Duration / time.Millisecond),
		"delay":        int(state.Delay / time.Millisecond),
		"delayJitter":  state.DelayJitter,
		"methods":      []string{state.ClassName + "#" + state.MethodName},
	}
	return commonPrepareEnd(config, state.AttackState, request)
}

// Start is called to start the action
// You can mutate the state here.
// You can use the result to return messages/errors/metrics or artifacts
func (l *javaMethodDelay) Start(_ context.Context, state *JavaMethodDelayState) (*action_kit_api.StartResult, error) {
	result, err := commonStart(state.AttackState)
	if err != nil {
		return result, err
	}
	if state.Validate {
		status := attack.GetAttackStatus(state.Pid)
		if status.AdviceApplied != "APPLIED" {
			return &action_kit_api.StartResult{
				Error: extutil.Ptr(action_kit_api.ActionKitError{
					Title:  "The given class and method did not match anything in the target JVM.",
					Status: extutil.Ptr(action_kit_api.Failed),
				}),
			}, nil
		}
	}
	return result, err
}

// Stop is called to stop the action
// It will be called even if the start method did not complete successfully.
// It should be implemented in a immutable way, as the agent might to retries if the stop method timeouts.
// You can use the result to return messages/errors/metrics or artifacts
func (l *javaMethodDelay) Stop(_ context.Context, state *JavaMethodDelayState) (*action_kit_api.StopResult, error) {
	return commonStop(state.AttackState)
}
