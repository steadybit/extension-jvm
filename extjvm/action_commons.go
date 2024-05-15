package extjvm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
	"github.com/steadybit/extension-jvm/extjvm/attachment"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/steadybit/extension-jvm/extjvm/jvmhttp"
	extension_kit "github.com/steadybit/extension-kit"
	"github.com/steadybit/extension-kit/extutil"
	"time"
)

var (
	erroneousCallRate = action_kit_api.ActionParameter{
		Name:         "erroneousCallRate",
		Label:        "Erroneous Call Rate",
		Description:  extutil.Ptr("How many percent of requests should trigger an exception?"),
		Type:         action_kit_api.Percentage,
		MinValue:     extutil.Ptr(0),
		MaxValue:     extutil.Ptr(100),
		DefaultValue: extutil.Ptr("100"),
		Required:     extutil.Ptr(true),
		Advanced:     extutil.Ptr(true),
	}
)

type JavaagentActionState struct {
	Duration     time.Duration
	Pid          int32
	ConfigJson   string
	EndpointPort int
	CallbackUrl  string
}

type javaagentAction struct {
	pluginJar      string
	description    action_kit_api.ActionDescription
	configProvider func(request action_kit_api.PrepareActionRequestBody) (any, error)
}

var (
	_ action_kit_sdk.Action[JavaagentActionState]         = (*javaagentAction)(nil)
	_ action_kit_sdk.ActionWithStop[JavaagentActionState] = (*javaagentAction)(nil)
)

func (j *javaagentAction) Describe() action_kit_api.ActionDescription {
	return j.description
}

func (j *javaagentAction) NewEmptyState() JavaagentActionState {
	return JavaagentActionState{}
}

func (j *javaagentAction) Prepare(_ context.Context, state *JavaagentActionState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	if duration, err := extractDuration(request); err == nil {
		state.Duration = duration
	} else {
		return nil, err
	}

	if pid, err := extractPid(request); err == nil {
		state.Pid = pid
	} else {
		return nil, err
	}

	if request.ExecutionContext != nil && request.ExecutionContext.AgentPid != nil && int(state.Pid) == *request.ExecutionContext.AgentPid {
		return nil, errors.New("can't attack the agent process")
	}

	config, err := j.configProvider(request)
	if err != nil {
		return nil, err
	}

	configJson, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	state.ConfigJson = string(configJson)
	javaVm := getJvm(state.Pid)
	if javaVm == nil {
		return nil, fmt.Errorf("jvm with pid %d not found", state.Pid)
	}

	callbackUrl, attackEndpointPort := startAttackEndpoint(javaVm, state.ConfigJson)
	state.EndpointPort = attackEndpointPort
	state.CallbackUrl = callbackUrl
	return nil, nil
}

func startAttackEndpoint(jvm *jvm.JavaVm, configJson string) (string, int) {
	attackEndpointPort := jvmhttp.StartAttackHttpServer(jvm.Pid, configJson)
	// The callback URL is used to send the attack results back to the agent.
	host := attachment.GetAttachment(jvm).GetAgentHost()
	callbackUrl := fmt.Sprintf("http://%s:%d", host, attackEndpointPort)
	log.Debug().Msgf("Callback URL: %s", callbackUrl)
	return callbackUrl, attackEndpointPort
}

func (j *javaagentAction) Start(_ context.Context, state *JavaagentActionState) (*action_kit_api.StartResult, error) {
	vm := getJvm(state.Pid)
	if vm == nil {
		return nil, extension_kit.ToError("VM not found", nil)
	}

	if err := start(vm, j.pluginJar, state.CallbackUrl); err != nil {
		return &action_kit_api.StartResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Failed to start action",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}, err
	}
	return nil, nil
}

func (j *javaagentAction) Stop(ctx context.Context, state *JavaagentActionState) (*action_kit_api.StopResult, error) {
	vm := getJvm(state.Pid)
	if vm == nil {
		return nil, extension_kit.ToError("VM not found", nil)
	}
	success := stop(vm, j.pluginJar)
	if !success {
		return &action_kit_api.StopResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Failed to stop action",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}, nil
	}
	return nil, nil
}

func extractDuration(request action_kit_api.PrepareActionRequestBody) (time.Duration, error) {
	parsedDuration := extutil.ToUInt64(request.Config["duration"])
	if parsedDuration == 0 {
		return 0, errors.New("duration is required")
	}
	return time.Duration(parsedDuration) * time.Millisecond, nil
}

func extractPid(request action_kit_api.PrepareActionRequestBody) (int32, error) {
	pids := request.Target.Attributes["process.pid"]
	if len(pids) == 0 {
		return 0, errors.New("attribute 'process.pid' is required")
	}
	return extutil.ToInt32(pids[0]), nil
}
