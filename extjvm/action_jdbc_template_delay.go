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

type jdbcTemplateDelay struct{}

type JdbcTemplateDelayState struct {
	Delay       time.Duration
	DelayJitter bool
	Operations  string
	JdbcUrl     string
	*AttackState
}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[JdbcTemplateDelayState]         = (*jdbcTemplateDelay)(nil)
	_ action_kit_sdk.ActionWithStop[JdbcTemplateDelayState] = (*jdbcTemplateDelay)(nil)
)

func NewJdbcTemplateDelay() action_kit_sdk.Action[JdbcTemplateDelayState] {
	return &jdbcTemplateDelay{}
}

func (l *jdbcTemplateDelay) NewEmptyState() JdbcTemplateDelayState {
	return JdbcTemplateDelayState{
		AttackState: &AttackState{},
	}
}

// Describe returns the action description for the platform with all required information.
func (l *jdbcTemplateDelay) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          TargetID + ".spring-jdbctemplate-delay-attack",
		Label:       "JDBC Template Delay",
		Description: "Delay a Spring JDBC Template response by the given duration.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(jdbcTemplateDelayIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			// The target type this action is for
			TargetType: targetIDOld + "(application.type=spring;spring.jdbc-template)",
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
				Name:         "operations",
				Label:        "Operation",
				Description:  extutil.Ptr("Which operation should be attacked?"),
				Type:         action_kit_api.String,
				DefaultValue: extutil.Ptr("*"),
				Required:     extutil.Ptr(true),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "Any",
						Value: "*",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Reads",
						Value: "r",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Writes",
						Value: "w",
					},
				}),
			},
			{
				Name:         "duration",
				Label:        "Duration",
				Description:  extutil.Ptr("How long should the traffic be dropped?"),
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
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
				Name:         "delayJitter",
				Label:        "Jitter",
				Description:  extutil.Ptr("Add random +/-30% jitter to response delay?"),
				Type:         action_kit_api.Boolean,
				DefaultValue: extutil.Ptr("true"),
				Required:     extutil.Ptr(true),
				Advanced:     extutil.Ptr(true),
			},
			{
				Name:         "jdbcUrl",
				Label:        "JDBC connection url",
				Description:  extutil.Ptr("Which JDBC connection should be attacked?"),
				Type:         action_kit_api.String,
				DefaultValue: extutil.Ptr("*"),
				Required:     extutil.Ptr(true),
				Advanced:     extutil.Ptr(true),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "Any",
						Value: "*",
					},
					action_kit_api.ParameterOptionsFromTargetAttribute{
						Attribute: "datasource.jdbc-url",
					},
				}),
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
func (l *jdbcTemplateDelay) Prepare(_ context.Context, state *JdbcTemplateDelayState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	state.JdbcUrl = extutil.ToString(request.Config["jdbcUrl"])
	state.Operations = extutil.ToString(request.Config["operations"])
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

	errResult := extractDuration(request, state.AttackState)
	if errResult != nil {
		return errResult, nil
	}

	errResult = extractPid(request, state.AttackState)
	if errResult != nil {
		return errResult, nil
	}

	var config = map[string]interface{}{
		"attack-class": "com.steadybit.attacks.javaagent.instrumentation.SpringJdbcTemplateDelayInstrumentation",
		"duration":     int(state.Duration / time.Millisecond),
		"delay":        int(state.Delay / time.Millisecond),
		"delayJitter":  state.DelayJitter,
		"operations":   state.Operations,
		"jdbc-url":     state.JdbcUrl,
	}
	return commonPrepareEnd(config, state.AttackState, request)
}

// Start is called to start the action
// You can mutate the state here.
// You can use the result to return messages/errors/metrics or artifacts
func (l *jdbcTemplateDelay) Start(_ context.Context, state *JdbcTemplateDelayState) (*action_kit_api.StartResult, error) {
	return commonStart(state.AttackState)
}

// Stop is called to stop the action
// It will be called even if the start method did not complete successfully.
// It should be implemented in a immutable way, as the agent might to retries if the stop method timeouts.
// You can use the result to return messages/errors/metrics or artifacts
func (l *jdbcTemplateDelay) Stop(_ context.Context, state *JdbcTemplateDelayState) (*action_kit_api.StopResult, error) {
	return commonStop(state.AttackState)
}
