package extjvm

import (
	"context"
	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_controlleException_Prepare(t *testing.T) {
	tests := []struct {
		name        string
		requestBody action_kit_api.PrepareActionRequestBody
		wantedError error
		wantedState *ControllerExceptionState
	}{
		{
			name: "Should return config",
			requestBody: action_kit_api.PrepareActionRequestBody{
				Config: map[string]interface{}{
					"action":            "prepare",
					"pattern":           "/customers",
					"method":            "GET",
					"duration":          "10000",
					"erroneousCallRate": 75,
				},
				ExecutionId: uuid.New(),
				Target: extutil.Ptr(action_kit_api.Target{
					Attributes: map[string][]string{
						"process.pid": {"42"},
					},
				}),
			},

			wantedState: &ControllerExceptionState{
				ControllerState: &ControllerState{
					AttackState: &AttackState{
						ConfigJson: "{\"attack-class\":\"com.steadybit.attacks.javaagent.instrumentation.JavaMethodExceptionInstrumentation\",\"duration\":10000,\"erroneousCallRate\":75,\"methods\":[\"com.steadybit.demo.CustomerController#customers\"]}",
					},
				},
			},
		},
	}
	action := NewControllerException()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//Given
			state := action.NewEmptyState()
			request := tt.requestBody
	    InitTestJVM()

			//When
			result, err := action.Prepare(context.Background(), &state, request)

			//Then
			if tt.wantedError != nil && err != nil {
				assert.EqualError(t, err, tt.wantedError.Error())
			} else if tt.wantedError != nil && result != nil {
				assert.Equal(t, result.Error.Title, tt.wantedError.Error())
			}
			if tt.wantedState != nil {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantedState.ConfigJson, state.ConfigJson)
			}
		})
	}
}
