package extjvm

import (
	"context"
	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_http_Client_Delay_Prepare(t *testing.T) {
	tests := []struct {
		name        string
		requestBody action_kit_api.PrepareActionRequestBody
		wantedState *HttpClientDelayState
	}{
		{
			name: "Should return config",
			requestBody: action_kit_api.PrepareActionRequestBody{
				Config: map[string]interface{}{
					"action":      "prepare",
					"hostAddress": "*",
					"duration":    "10000",
					"delay":       "500",
					"delayJitter": "true",
				},
				ExecutionId: uuid.New(),
				Target: extutil.Ptr(action_kit_api.Target{
					Attributes: map[string][]string{
						"process.pid": {"42"},
					},
				}),
			},

			wantedState: &HttpClientDelayState{
				AttackState: &AttackState{
					ConfigJson: "{\"attack-class\":\"com.steadybit.attacks.javaagent.instrumentation.SpringHttpClientDelayInstrumentation\",\"delay\":500,\"delayJitter\":true,\"duration\":10000,\"hostAddress\":\"*\"}",
				},
			},
		},
	}
	action := NewHttpClientDelay()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//Given
			state := action.NewEmptyState()
			request := tt.requestBody
			InitTestJVM()

			//When
			action.Prepare(context.Background(), &state, request)
			//Then
			if tt.wantedState != nil {
				assert.Equal(t, tt.wantedState.ConfigJson, state.ConfigJson)
			}
		})
	}
}
