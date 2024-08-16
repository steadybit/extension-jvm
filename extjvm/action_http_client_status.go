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

type httpClientStatus struct{}

type HttpClientStatusState struct {
	ErroneousCallRate int
	HttpMethods       []string
	HostAddress       string
	UrlPath           string
	FailureCauses     []string
	*AttackState
}

// Make sure action implements all required interfaces
var (
	_ action_kit_sdk.Action[HttpClientStatusState]         = (*httpClientStatus)(nil)
	_ action_kit_sdk.ActionWithStop[HttpClientStatusState] = (*httpClientStatus)(nil)
)

func NewHttpClientStatus() action_kit_sdk.Action[HttpClientStatusState] {
	return &httpClientStatus{}
}

func (l *httpClientStatus) NewEmptyState() HttpClientStatusState {
	return HttpClientStatusState{
		AttackState: &AttackState{},
	}
}

// Describe returns the action description for the platform with all required information.
func (l *httpClientStatus) Describe() action_kit_api.ActionDescription {
	return action_kit_api.ActionDescription{
		Id:          ActionIDPrefix + ".spring-httpclient-status-attack",
		Label:       "Http Client Status",
		Description: "Returns the given status code for a RestTemplate or WebClient call. The original call is not executed.",
		Version:     extbuild.GetSemverVersionStringOrUnknown(),
		Icon:        extutil.Ptr(springHttpStatusIcon),
		TargetSelection: extutil.Ptr(action_kit_api.TargetSelection{
			// The target type this action is for
			TargetType: targetType + "(instance.type=spring)",
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
				Name:         "duration",
				Label:        "Duration",
				Description:  extutil.Ptr("How long should the calls be attacked?"),
				Type:         action_kit_api.Duration,
				DefaultValue: extutil.Ptr("30s"),
				Required:     extutil.Ptr(true),
			},
			{
				Name:        "httpMethods",
				Label:       "HttpMethods",
				Description: extutil.Ptr("Which HTTP methods should be attacked?"),
				Type:        action_kit_api.StringArray,
				Required:    extutil.Ptr(false),
				Advanced:    extutil.Ptr(true),
				Options:     methodsOptions,
			},
			{
				Name:         "hostAddress",
				Label:        "Host Address",
				Description:  extutil.Ptr("Request to which host address should be attacked?"),
				Type:         action_kit_api.String,
				DefaultValue: extutil.Ptr("*"),
				Required:     extutil.Ptr(false),
				Advanced:     extutil.Ptr(true),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "Any",
						Value: "*",
					},
					action_kit_api.ParameterOptionsFromTargetAttribute{
						Attribute: "spring-instance.http-outgoing-calls",
					},
				}),
			},
			{
				Name:         "urlPath",
				Label:        "URL Path",
				Description:  extutil.Ptr("Which URL paths should be attacked?"),
				Type:         action_kit_api.String,
				DefaultValue: extutil.Ptr("*"),
				Required:     extutil.Ptr(false),
				Advanced:     extutil.Ptr(true),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "Any",
						Value: "*",
					},
				}),
			},
			{
				Name:        "failureCauses",
				Label:       "Failure Types",
				Description: extutil.Ptr("What HTTP client behavior should be simulated? Will overwrite any HttpStatus configuration."),
				Type:        action_kit_api.StringArray,
				Required:    extutil.Ptr(false),
				Advanced:    extutil.Ptr(true),
				Options: extutil.Ptr([]action_kit_api.ParameterOption{
					action_kit_api.ExplicitParameterOption{
						Label: "Protocol & Network Errors",
						Value: "ERROR",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Request Timeouts",
						Value: "TIMEOUT",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with 500 status code",
						Value: "HTTP_500",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with 502 status code",
						Value: "HTTP_502",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with 503 status code",
						Value: "HTTP_503",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with 504 status code",
						Value: "HTTP_504",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with a random 5XX status code",
						Value: "HTTP_5XX",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with 400 status code",
						Value: "HTTP_400",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with 403 status code",
						Value: "HTTP_403",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with 404 status code",
						Value: "HTTP_404",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with 429 status code",
						Value: "HTTP_429",
					},
					action_kit_api.ExplicitParameterOption{
						Label: "Response with a random 4XX status code",
						Value: "HTTP_4XX",
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
func (l *httpClientStatus) Prepare(_ context.Context, state *HttpClientStatusState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {

	errResult := extractDuration(request, state.AttackState)
	if errResult != nil {
		return errResult, nil
	}
	state.ErroneousCallRate = extutil.ToInt(request.Config["erroneousCallRate"])

	state.HttpMethods = extutil.ToStringArray(request.Config["httpMethods"])
	state.HostAddress = extutil.ToString(request.Config["hostAddress"])
	state.UrlPath = extutil.ToString(request.Config["urlPath"])
	state.FailureCauses = extutil.ToStringArray(request.Config["failureCauses"])

	errResult = extractPid(request, state.AttackState)
	if errResult != nil {
		return errResult, nil
	}

	var config = map[string]interface{}{
		"attack-class":      "com.steadybit.attacks.javaagent.instrumentation.SpringHttpClientStatusInstrumentation",
		"duration":          int(state.Duration / time.Millisecond),
		"erroneousCallRate": state.ErroneousCallRate,
		"httpMethods":       state.HttpMethods,
		"hostAddress":       state.HostAddress,
		"urlPath":           state.UrlPath,
		"failureCauses":     state.FailureCauses,
	}
	return commonPrepareEnd(config, state.AttackState, request)
}

// Start is called to start the action
// You can mutate the state here.
// You can use the result to return messages/errors/metrics or artifacts
func (l *httpClientStatus) Start(_ context.Context, state *HttpClientStatusState) (*action_kit_api.StartResult, error) {
	return commonStart(state.AttackState)
}

// Stop is called to stop the action
// It will be called even if the start method did not complete successfully.
// It should be implemented in a immutable way, as the agent might to retries if the stop method timeouts.
// You can use the result to return messages/errors/metrics or artifacts
func (l *httpClientStatus) Stop(_ context.Context, state *HttpClientStatusState) (*action_kit_api.StopResult, error) {
	return commonStop(state.AttackState)
}
