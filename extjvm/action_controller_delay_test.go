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

func Test_controllerDelay_Prepare(t *testing.T) {
	facade := &mockJavaFacade{}
	spring := &SpringDiscovery{}

	fake, err := facade.startFakeJvm()
	require.NoError(t, err)
	defer func(fake *FakeJvm) {
		_ = fake.stop()
	}(fake)

	spring.applications.Store(fake.Pid(), SpringApplication{
		Name: "customers",
		Pid:  fake.Pid(),
		MvcMappings: []SpringMvcMapping{
			{
				Methods:      []string{"GET"},
				Patterns:     []string{"/customers"},
				HandlerClass: "com.steadybit.demo.CustomerController",
				HandlerName:  "customers",
			},
		},
	})

	tests := []struct {
		name        string
		requestBody action_kit_api.PrepareActionRequestBody

		wantedState *JavaagentActionState
	}{
		{
			name: "Should return config with deprecated method parameter",
			requestBody: action_kit_api.PrepareActionRequestBody{
				Config: map[string]interface{}{
					"action":      "prepare",
					"pattern":     "/customers",
					"method":      "GET",
					"methods":     []interface{}{"POST"},
					"duration":    "10000",
					"delay":       "500",
					"delayJitter": "true",
				},
				ExecutionId: uuid.New(),
				Target:      extutil.Ptr(fake.getTarget()),
			},

			wantedState: &JavaagentActionState{
				ConfigJson: "{\"attack-class\":\"com.steadybit.attacks.javaagent.instrumentation.JavaMethodDelayInstrumentation\",\"delay\":500,\"delayJitter\":true,\"duration\":10000,\"methods\":[\"com.steadybit.demo.CustomerController#customers\"]}",
			},
		},
		{
			name: "Should return config",
			requestBody: action_kit_api.PrepareActionRequestBody{
				Config: map[string]interface{}{
					"action":      "prepare",
					"pattern":     "/customers",
					"methods":     []interface{}{"GET"},
					"duration":    "10000",
					"delay":       "500",
					"delayJitter": "true",
				},
				ExecutionId: uuid.New(),
				Target:      extutil.Ptr(fake.getTarget()),
			},

			wantedState: &JavaagentActionState{
				ConfigJson: "{\"attack-class\":\"com.steadybit.attacks.javaagent.instrumentation.JavaMethodDelayInstrumentation\",\"delay\":500,\"delayJitter\":true,\"duration\":10000,\"methods\":[\"com.steadybit.demo.CustomerController#customers\"]}",
			},
		},
	}
	action := NewControllerDelay(facade, spring)

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
				assert.Equal(t, tt.wantedState.ConfigJson, state.ConfigJson)
			}
		})
	}
}
