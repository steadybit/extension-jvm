package extjvm

import (
	"context"
	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_JDBC_Template_Delay_Prepare(t *testing.T) {
	tests := []struct {
		name        string
		requestBody action_kit_api.PrepareActionRequestBody
		wantedState *JdbcTemplateDelayState
	}{
		{
			name: "Should return config",
			requestBody: action_kit_api.PrepareActionRequestBody{
				Config: map[string]interface{}{
					"action":      "prepare",
					"jdbcUrl":     "jdbc:mysql://localhost:3306/test",
					"operations":  "w",
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

			wantedState: &JdbcTemplateDelayState{
				AttackState: &AttackState{
					ConfigJson: "{\"attack-class\":\"com.steadybit.attacks.javaagent.instrumentation.SpringJdbcTemplateDelayInstrumentation\",\"delay\":500,\"delayJitter\":true,\"duration\":10000,\"jdbc-url\":\"jdbc:mysql://localhost:3306/test\",\"operations\":\"w\"}",
				},
			},
		},
	}
	action := NewJdbcTemplateDelay()
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
