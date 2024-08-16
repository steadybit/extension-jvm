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

type jdbcTemplateException struct{}

type JdbcTemplateExceptionState struct {
	ErroneousCallRate int
	Operations        string
	JdbcUrl           string
	*AttackState
}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[JdbcTemplateExceptionState]         = (*jdbcTemplateException)(nil)
	_ action_kit_sdk.ActionWithStop[JdbcTemplateExceptionState] = (*jdbcTemplateException)(nil)
)

func NewJdbcTemplateException() action_kit_sdk.Action[JdbcTemplateExceptionState] {
	return &jdbcTemplateException{}
}

func (l *jdbcTemplateException) NewEmptyState() JdbcTemplateExceptionState {
	return JdbcTemplateExceptionState{
		AttackState: &AttackState{},
	}
}

// Describe returns the action description for the platform with all required information.
func (l *jdbcTemplateException) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          ActionIDPrefix + ".spring-jdbctemplate-exception-attack",
		Label:       "JDBC Template Exception",
		Description: "Throws an exception in a Spring JDBC Template.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(jdbcTemplateExceptionIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			// The target type this action is for
			TargetType: targetType + "(instance.type=spring;spring-instance.jdbc-template)",
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
func (l *jdbcTemplateException) Prepare(_ context.Context, state *JdbcTemplateExceptionState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	state.ErroneousCallRate = extutil.ToInt(request.Config["erroneousCallRate"])

	state.JdbcUrl = extutil.ToString(request.Config["jdbcUrl"])
	state.Operations = extutil.ToString(request.Config["operations"])

	errResult := extractDuration(request, state.AttackState)
	if errResult != nil {
		return errResult, nil
	}

	errResult = extractPid(request, state.AttackState)
	if errResult != nil {
		return errResult, nil
	}

	var config = map[string]interface{}{
		"attack-class":      "com.steadybit.attacks.javaagent.instrumentation.SpringJdbcTemplateExceptionInstrumentation",
		"duration":          int(state.Duration / time.Millisecond),
		"operations":        state.Operations,
		"erroneousCallRate": state.ErroneousCallRate,
		"jdbc-url":          state.JdbcUrl,
	}
	return commonPrepareEnd(config, state.AttackState, request)
}

// Start is called to start the action
// You can mutate the state here.
// You can use the result to return messages/errors/metrics or artifacts
func (l *jdbcTemplateException) Start(_ context.Context, state *JdbcTemplateExceptionState) (*action_kit_api.StartResult, error) {
	return commonStart(state.AttackState)
}

// Stop is called to stop the action
// It will be called even if the start method did not complete successfully.
// It should be implemented in a immutable way, as the agent might to retries if the stop method timeouts.
// You can use the result to return messages/errors/metrics or artifacts
func (l *jdbcTemplateException) Stop(_ context.Context, state *JdbcTemplateExceptionState) (*action_kit_api.StopResult, error) {
	return commonStop(state.AttackState)
}
