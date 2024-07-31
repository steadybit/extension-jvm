/*
 * Copyright 2023 steadybit GmbH. All rights reserved.
 */

package extjvm

import (
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"github.com/steadybit/extension-kit/extbuild"
	"github.com/steadybit/extension-kit/extutil"
	"time"
)

func NewJdbcTemplateDelay(facade jvm.JavaFacade) action_kit_sdk.Action[JavaagentActionState] {
	return &javaagentAction{
		pluginJar:      utils.GetJarPath("attack-spring-javaagent.jar"),
		description:    jdbcTemplateDelayDescribe(),
		configProvider: jdbcTemplateDelayConfigProvider,
		facade:         facade,
	}
}

func jdbcTemplateDelayDescribe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          ActionIDPrefix + ".spring-jdbctemplate-delay-attack",
		Label:       "JDBC Template Delay",
		Description: "Delay a Spring JDBC Template response by the given duration.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(jdbcTemplateDelayIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			// The target type this action is for
			TargetType: targetID + "(instance.type=spring;spring-instance.jdbc-template)",
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

func jdbcTemplateDelayConfigProvider(request action_kit_api.PrepareActionRequestBody) (any, error) {
	duration, err := extractDuration(request)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"attack-class": "com.steadybit.attacks.spring.instrumentation.SpringJdbcTemplateDelayInstrumentation",
		"duration":     int(duration / time.Millisecond),
		"delay":        extutil.ToUInt64(request.Config["delay"]),
		"delayJitter":  extutil.ToBool(request.Config["delayJitter"]),
		"operations":   extutil.ToString(request.Config["operations"]),
		"jdbc-url":     extutil.ToString(request.Config["jdbcUrl"]),
	}, nil
}
