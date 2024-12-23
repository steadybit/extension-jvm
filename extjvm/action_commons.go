package extjvm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_sdk"
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
	targetSelectionTemplates = []action_kit_api.TargetSelectionTemplate{
		{
			Label:       "instance name",
			Description: extutil.Ptr("Find instance by name."),
			Query:       "jvm-instance.name=\"\"",
		},
	}
)

type JavaagentActionState struct {
	Duration              time.Duration
	Pid                   int32
	ConfigJson            string
	EndpointPort          int
	CallbackUrl           string
	ValidateAdviceApplied bool
}

type javaagentAction struct {
	pluginJar      string
	description    action_kit_api.ActionDescription
	configProvider func(request action_kit_api.PrepareActionRequestBody) (map[string]interface{}, error)
	facade         jvm.JavaFacade
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
	javaVm := j.facade.GetJvm(state.Pid)
	if javaVm == nil {
		return nil, fmt.Errorf("jvm with pid %d not found", state.Pid)
	}

	state.ValidateAdviceApplied = extutil.ToBool(request.Config["validate"])

	callbackUrl, attackEndpointPort := startAttackEndpoint(javaVm, state.ConfigJson)
	state.EndpointPort = attackEndpointPort
	state.CallbackUrl = callbackUrl
	return nil, nil
}

func startAttackEndpoint(javaVm jvm.JavaVm, configJson string) (string, int) {
	attackEndpointPort := jvmhttp.StartAttackHttpServer(javaVm.Pid(), configJson)
	// The callback URL is used to send the attack results back to the agent.
	host := jvm.GetAttachment(javaVm).GetHostAddress()
	callbackUrl := fmt.Sprintf("http://%s:%d", host, attackEndpointPort)
	log.Debug().Msgf("Callback URL: %s", callbackUrl)
	return callbackUrl, attackEndpointPort
}

func (j *javaagentAction) Start(_ context.Context, state *JavaagentActionState) (*action_kit_api.StartResult, error) {
	javaVm := j.facade.GetJvm(state.Pid)
	if javaVm == nil {
		return nil, extension_kit.ToError("JVM not found", nil)
	}

	if status, err := j.startAttack(javaVm, j.pluginJar, state.CallbackUrl); err != nil {
		return nil, extension_kit.ToError("Failed to start action", err)
	} else if state.ValidateAdviceApplied && status.AdviceApplied != "APPLIED" {
		return &action_kit_api.StartResult{
			Error: extutil.Ptr(action_kit_api.ActionKitError{
				Title:  "The given class and method did not match anything in the target JVM.",
				Status: extutil.Ptr(action_kit_api.Failed),
			}),
		}, nil
	}
	return &action_kit_api.StartResult{
		Messages: &[]action_kit_api.Message{{
			Message: fmt.Sprintf("Action on PID %d started using %s", state.Pid, state.ConfigJson),
		}},
	}, nil
}

func (j *javaagentAction) Stop(_ context.Context, state *JavaagentActionState) (*action_kit_api.StopResult, error) {
	javaVm := j.facade.GetJvm(state.Pid)
	var msg action_kit_api.Message
	if javaVm != nil {
		if err := j.stopAttack(javaVm, j.pluginJar); err != nil {
			return nil, extension_kit.ToError("Failed to stop action", nil)
		}
		msg.Level = extutil.Ptr(action_kit_api.Info)
		msg.Message = fmt.Sprintf("Action on PID %d stopped", state.Pid)
	} else {
		msg.Level = extutil.Ptr(action_kit_api.Warn)
		msg.Message = "JVM not found - skipping stop action"
	}
	return &action_kit_api.StopResult{
		Messages: &[]action_kit_api.Message{msg},
	}, nil
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

var (
	attackStartTimeout = 10 * time.Second
)

func (j *javaagentAction) startAttack(javaVm jvm.JavaVm, pluginJar, callbackUrl string) (*jvmhttp.Status, error) {
	if err := j.facade.LoadAgentPlugin(javaVm, pluginJar, callbackUrl); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), attackStartTimeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timed out waiting for Java Attack instrumentation after %s", attackStartTimeout)
		case <-time.After(100 * time.Millisecond):
			status := jvmhttp.GetAttackStatus(javaVm.Pid())
			if status.Started {
				return &status, nil
			} else if status.Failure != "" {
				return &status, fmt.Errorf("failed to start attack: %s", status.Failure)
			}
		}
	}
}

func (j *javaagentAction) stopAttack(javaVm jvm.JavaVm, pluginJar string) error {
	jvmhttp.StopAttackHttpServer(javaVm.Pid())
	return j.facade.UnloadAgentPlugin(javaVm, pluginJar)
}
