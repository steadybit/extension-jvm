package extjvm

import (
  "github.com/steadybit/action-kit/go/action_kit_api/v2"
  "github.com/steadybit/extension-kit/extutil"
  "time"
)

func extractDuration(request action_kit_api.PrepareActionRequestBody, state *ControllerState) *action_kit_api.PrepareResult {
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

func extractPid(request action_kit_api.PrepareActionRequestBody, state *ControllerState) *action_kit_api.PrepareResult {
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
