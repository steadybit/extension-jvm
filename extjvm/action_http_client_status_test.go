// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

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

func Test_http_Client_Status_Prepare(t *testing.T) {
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
					"action":            "prepare",
					"erroneousCallRate": 75,
					"duration":          "10000",
					"httpMethods":       []interface{}{"GET"},
					"hostAddress":       "*",
					"urlPath":           "/test",
					"failureCauses":     []interface{}{"HTTP_502"},
				},
				ExecutionId: uuid.New(),
				Target:      extutil.Ptr(fake.getTarget()),
			},

			wantedState: &JavaagentActionState{
				ConfigJson: "{\"attack-class\":\"com.steadybit.attacks.javaagent.instrumentation.SpringHttpClientStatusInstrumentation\",\"duration\":10000,\"erroneousCallRate\":75,\"failureCauses\":[\"HTTP_502\"],\"hostAddress\":\"*\",\"httpMethods\":[\"GET\"],\"urlPath\":\"/test\"}",
			},
		},
	}
	action := NewHttpClientStatus(facade)
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
