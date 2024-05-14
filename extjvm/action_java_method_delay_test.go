package extjvm

import (
	"context"
	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_Java_Method_Delay_Prepare(t *testing.T) {
	facade := &mockJavaFacade{}
	fake, err := facade.startFakeJvm()
	require.NoError(t, err)
	defer func(fake *FakeJvm) {
		_ = fake.stop()
	}(fake)

	tests := []struct {
		name        string
		requestBody action_kit_api.PrepareActionRequestBody
		wantedState *JavaagentActionState
	}{
		{
			name: "Should return config",
			requestBody: action_kit_api.PrepareActionRequestBody{
				Config: map[string]interface{}{
					"action":      "prepare",
					"className":   "com.steadybit.demo.CustomerController",
					"methodName":  "GetCustomers",
					"duration":    "10000",
					"delay":       "500",
					"delayJitter": "true",
					"validate":    "true",
				},
				ExecutionId: uuid.New(),
				Target:      extutil.Ptr(fake.getTarget()),
			},

			wantedState: &JavaagentActionState{
				ValidateAdviceApplied: true,
				ConfigJson:            "{\"attack-class\":\"com.steadybit.attacks.javaagent.instrumentation.JavaMethodDelayInstrumentation\",\"delay\":500,\"delayJitter\":true,\"duration\":10000,\"methods\":[\"com.steadybit.demo.CustomerController#GetCustomers\"]}",
			},
		},
	}
	action := NewJavaMethodDelay(facade)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			//Given
			state := action.NewEmptyState()
			request := tt.requestBody

			//When
			_, err := action.Prepare(context.Background(), &state, request)
			assert.NoError(t, err)
			//Then
			if tt.wantedState != nil {
				assert.Equal(t, tt.wantedState.ValidateAdviceApplied, state.ValidateAdviceApplied)
				assert.Equal(t, tt.wantedState.ConfigJson, state.ConfigJson)
			}
		})
	}
}
