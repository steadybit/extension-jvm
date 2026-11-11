// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2026 Steadybit GmbH

package extjvm

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_Start_reflects_status_reported_by_javaagent(t *testing.T) {
	tests := []struct {
		name          string
		validate      bool
		callbackPath  string
		callbackBody  string
		wantErr       string
		wantFailTitle string
	}{
		{
			name:         "started with advice applied",
			validate:     true,
			callbackPath: "/started",
			callbackBody: "APPLIED",
		},
		{
			name:         "failed instrumentation fails the action",
			callbackPath: "/failed",
			callbackBody: "Failed to instrument com.steadybit.demo.CustomerController on Java 27",
			wantErr:      "Failed to instrument com.steadybit.demo.CustomerController on Java 27",
		},
		{
			name:          "advice not applied fails the action when validation is requested",
			validate:      true,
			callbackPath:  "/started",
			callbackBody:  "NOT_APPLIED",
			wantFailTitle: "The given class and method did not match anything in the target JVM.",
		},
		{
			name:         "advice not applied is tolerated without validation",
			callbackPath: "/started",
			callbackBody: "NOT_APPLIED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			facade := &mockJavaFacade{}
			fake, err := facade.startFakeJvm()
			require.NoError(t, err)
			defer func() { _ = fake.stop() }()

			facade.On("LoadAgentPlugin", fake, mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
				resp, err := http.Post(args.String(2)+tt.callbackPath, "text/plain", strings.NewReader(tt.callbackBody))
				require.NoError(t, err)
				_ = resp.Body.Close()
			})
			facade.On("UnloadAgentPlugin", fake, mock.Anything).Return(nil)

			action := NewJavaMethodException(facade)
			state := action.NewEmptyState()
			_, err = action.Prepare(context.Background(), &state, action_kit_api.PrepareActionRequestBody{
				Config: map[string]any{
					"className":         "com.steadybit.demo.CustomerController",
					"methodName":        "GetCustomers",
					"duration":          "10000",
					"erroneousCallRate": 100,
					"validate":          tt.validate,
				},
				ExecutionId: uuid.New(),
				Target:      new(fake.getTarget()),
			})
			require.NoError(t, err)
			defer func() {
				_, _ = action.(*javaagentAction).Stop(context.Background(), &state)
			}()

			result, err := action.Start(context.Background(), &state)

			switch {
			case tt.wantErr != "":
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			case tt.wantFailTitle != "":
				require.NoError(t, err)
				require.NotNil(t, result.Error)
				assert.Equal(t, tt.wantFailTitle, result.Error.Title)
			default:
				require.NoError(t, err)
				assert.Nil(t, result.Error)
			}
		})
	}
}
