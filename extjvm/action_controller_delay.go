/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extjvm

import (
  "context"
  "encoding/json"
  "github.com/google/uuid"
  "github.com/steadybit/action-kit/go/action_kit_api/v2"
  "github.com/steadybit/action-kit/go/action_kit_sdk"
  extension_kit "github.com/steadybit/extension-kit"
  "github.com/steadybit/extension-kit/extbuild"
  "github.com/steadybit/extension-kit/extutil"
  "time"
)

type controllerDelay struct{}

type ControllerDelayState struct {
	ExecutionID  uuid.UUID
	Delay        time.Duration
	Duration     time.Duration
	Pattern      string
	Method       string
	DelayJitter  bool
	Pid          int32
	ConfigJson   string
	EndpointPort int
	CallbackUrl  string
}

type ExecutionRunData struct {
	stopAction chan bool // stores the stop channels for each execution
}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[ControllerDelayState]         = (*controllerDelay)(nil)
	_ action_kit_sdk.ActionWithStop[ControllerDelayState] = (*controllerDelay)(nil) // Optional, needed when the action needs a stop method

)

func NewControllerDelay() action_kit_sdk.Action[ControllerDelayState] {
	return &controllerDelay{}
}

func (l *controllerDelay) NewEmptyState() ControllerDelayState {
	return ControllerDelayState{}
}

// Describe returns the action description for the platform with all required information.
func (l *controllerDelay) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          targetID + ".spring-mvc-delay-attack",
		Label:       "Controller Delay",
		Description: "Delay a Spring MVC controller http response by the given duration.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(controllerDelayIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			// The target type this action is for
			TargetType: targetID,
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
			{
				Name:        "pattern",
				Label:       "Request Mapping",
				Description: extutil.Ptr("Which request mapping pattern should be used to match the requests?"),
				Type:        action_kit_api.String,
				Required:    extutil.Ptr(true),
				Order:       extutil.Ptr(1),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ParameterOptionsFromTargetAttribute{
						Attribute: "spring.mvc-mapping",
					},
				}),
			},
			{
				Name:         "method",
				Label:        "Http Method",
				Description:  extutil.Ptr("Which http method should be used to match the requests?"),
				Type:         action_kit_api.String,
				Required:     extutil.Ptr(true),
				DefaultValue: extutil.Ptr("*"),
				Order:        extutil.Ptr(2),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "Any",
						Value: "*",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "GET",
						Value: "GET",
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
						Label: "HEAD",
						Value: "HEAD",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "DELETE",
						Value: "DELETE",
					},
				}),
			},
			{
				Name:         "delay",
				Label:        "Delay",
				Description:  extutil.Ptr("How much should the response be delayed?"),
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("500ms"),
				Required:     extutil.Ptr(true),
				Order:        extutil.Ptr(3),
			},
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  extutil.Ptr("How long should the traffic be dropped?"),
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
				Order:        extutil.Ptr(4),
			}, {
				Name:         "delayJitter",
				Label:        "Jitter",
				Description:  extutil.Ptr("Add random +/-30% jitter to response delay?"),
				Type:         action_kit_api.Boolean,
				DefaultValue: extutil.Ptr("true"),
				Required:     extutil.Ptr(true),
				Advanced:     extutil.Ptr(true),
				Order:        extutil.Ptr(5),
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
func (l *controllerDelay) Prepare(_ context.Context, state *ControllerDelayState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	pattern := extutil.ToString(request.Config["pattern"])
	if pattern == "" {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Pattern is required",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}, nil
	}
	state.Pattern = pattern

	method := extutil.ToString(request.Config["method"])
	if pattern == "" {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Method is required",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}, nil
	}
	state.Method = method

	parsedDelay := extutil.ToUInt64(request.Config["delay"])
	var delay time.Duration
	if parsedDelay == 0 {
		delay = 0
	} else {
		delay = time.Duration(parsedDelay) * time.Millisecond
	}
	state.Delay = delay

	parsedDuration := extutil.ToUInt64(request.Config["duration"])
	if parsedDuration == 0 {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Duration is required",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}, nil
	}
	duration := time.Duration(parsedDuration) * time.Millisecond
	state.Duration = duration

	delayJitter := extutil.ToBool(request.Config["delayJitter"])
	state.DelayJitter = delayJitter

	pids := request.Target.Attributes["process.pid"]
	if len(pids) == 0 {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Process pid is required",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}, nil
	}
	state.Pid = extutil.ToInt32(pids[0])

	var config = map[string]interface{}{
		"attack-class": "com.steadybit.attacks.javaagent.instrumentation.JavaMethodDelayInstrumentation",
		"duration":     state.Duration * time.Millisecond,
		"delay":        state.Delay * time.Millisecond,
		"delayJitter":  state.DelayJitter,
		"methods":      state.Method,
	}
	configJson, err := json.Marshal(config)
	if err != nil {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Failed to marshal config",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}, err
	}
  vm := GetTarget(state.Pid)
  if vm == nil {
    return nil, extension_kit.ToError("VM not found", nil)
  }
	callbackUrl, attackEndpointPort := Prepare(vm, string(configJson))
	state.EndpointPort = attackEndpointPort
	state.CallbackUrl = callbackUrl

	return nil, nil
}

// Start is called to start the action
// You can mutate the state here.
// You can use the result to return messages/errors/metrics or artifacts
func (l *controllerDelay) Start(_ context.Context, state *ControllerDelayState) (*action_kit_api.StartResult, error) {
  vm := GetTarget(state.Pid)
  if vm == nil {
    return nil, extension_kit.ToError("VM not found", nil)
  }
	err := Start(vm, state.CallbackUrl)
	if err != nil {
		return &action_kit_api.StartResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Failed to start attack",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}, err
	}
	return nil, nil
}

// Stop is called to stop the action
// It will be called even if the start method did not complete successfully.
// It should be implemented in a immutable way, as the agent might to retries if the stop method timeouts.
// You can use the result to return messages/errors/metrics or artifacts
func (l *controllerDelay) Stop(_ context.Context, state *ControllerDelayState) (*action_kit_api.StopResult, error) {
  vm := GetTarget(state.Pid)
  if vm == nil {
    return nil, extension_kit.ToError("VM not found", nil)
  }
	Stop(vm)
	return nil, nil
}
