package extjvm

import (
	"context"
	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Java_Method_Exception_Prepare(t *testing.T) {
	tests := []struct {
		name        string
		requestBody action_kit_api.PrepareActionRequestBody
		wantedState *JavaMethodExceptionState
	}{
		{
			name: "Should return config",
			requestBody: action_kit_api.PrepareActionRequestBody{
				Config: map[string]interface{}{
					"action":            "prepare",
					"className":         "com.steadybit.demo.CustomerController",
					"methodName":        "GetCustomers",
					"duration":          "10000",
					"erroneousCallRate": 75,
					"validate":          "true",
				},
				ExecutionId: uuid.New(),
				Target: extutil.Ptr(action_kit_api.Target{
					Attributes: map[string][]string{
						"process.pid": {"42"},
					},
				}),
			},

			wantedState: &JavaMethodExceptionState{
				ClassName:  "com.steadybit.demo.CustomerController",
				MethodName: "GetCustomers",
				Validate:   true,
				AttackState: &AttackState{
					ConfigJson: "{\"attack-class\":\"com.steadybit.attacks.javaagent.instrumentation.JavaMethodExceptionInstrumentation\",\"duration\":10000,\"erroneousCallRate\":75,\"methods\":[\"com.steadybit.demo.CustomerController#GetCustomers\"]}",
				},
			},
		},
	}
	action := NewJavaMethodException()
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
				assert.Equal(t, tt.wantedState.ClassName, state.ClassName)
				assert.Equal(t, tt.wantedState.MethodName, state.MethodName)
				assert.Equal(t, tt.wantedState.Validate, state.Validate)
				assert.Equal(t, tt.wantedState.ConfigJson, state.ConfigJson)
			}
		})
	}
}
