package extjvm

import (
	"encoding/json"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-jvm/extjvm/common"
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

	attackJavaJavaagentJar        = common.GetJarPath("attack-java-javaagent.jar")
	attackSpringBoot2JavaagentJar = common.GetJarPath("attack-springboot2-javaagent.jar")
)

type AttackState struct {
	Duration     time.Duration
	Pid          int32
	ConfigJson   string
	EndpointPort int
	CallbackUrl  string
}

func extractDuration(request action_kit_api.PrepareActionRequestBody, state *AttackState) *action_kit_api.PrepareResult {
	parsedDuration := extutil.ToUInt64(request.Config["duration"])
	if parsedDuration == 0 {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Duration is required",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}
	}
	duration := time.Duration(parsedDuration) * time.Millisecond
	state.Duration = duration
	return nil
}

func extractPid(request action_kit_api.PrepareActionRequestBody, state *AttackState) *action_kit_api.PrepareResult {
	pids := request.Target.Attributes["process.pid"]
	if len(pids) == 0 {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Process pid is required",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}
	}
	state.Pid = extutil.ToInt32(pids[0])
	return nil
}

func commonStart(state *AttackState, pluginJar string) (*action_kit_api.StartResult, error) {
	vm := GetTarget(state.Pid)
	if vm == nil {
		return nil, extension_kit.ToError("VM not found", nil)
	}
	err := Start(vm, pluginJar, state.CallbackUrl)
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

func commonStop(state *AttackState, pluginJar string) (*action_kit_api.StopResult, error) {
	vm := GetTarget(state.Pid)
	if vm == nil {
		return nil, extension_kit.ToError("VM not found", nil)
	}
	success := Stop(vm, pluginJar)
	if !success {
		return &action_kit_api.StopResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Failed to stop attack",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}, nil
	}
	return nil, nil
}

func commonPrepareEnd(config map[string]interface{}, state *AttackState, request action_kit_api.PrepareActionRequestBody) (*action_kit_api.PrepareResult, error) {
	if request.ExecutionContext != nil && request.ExecutionContext.AgentPid != nil && int(state.Pid) == *request.ExecutionContext.AgentPid {
		return &action_kit_api.PrepareResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "Can't attack the agent process",
				Status: extutil.Ptr(action_kit_api.Errored),
			}),
		}, nil
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
	state.ConfigJson = string(configJson)
	vm := GetTarget(state.Pid)
	if vm == nil {
		return nil, extension_kit.ToError("VM not found", nil)
	}
	callbackUrl, attackEndpointPort := Prepare(vm, string(configJson))
	state.EndpointPort = attackEndpointPort
	state.CallbackUrl = callbackUrl

	return nil, nil
}
