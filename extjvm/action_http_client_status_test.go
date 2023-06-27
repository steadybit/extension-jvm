package extjvm

import (
  "context"
  "github.com/google/uuid"
  "github.com/steadybit/action-kit/go/action_kit_api/v2"
  "github.com/steadybit/extension-kit/extutil"
  "github.com/stretchr/testify/assert"
  "testing"
)

func Test_http_Client_Status_Prepare(t *testing.T) {
  tests := []struct {
    name        string
    requestBody action_kit_api.PrepareActionRequestBody
    wantedError error
    wantedState *HttpClientStatusState
  }{
    {
      name: "Should return config",
      requestBody: action_kit_api.PrepareActionRequestBody{
        Config: map[string]interface{}{
          "action":      "prepare",
          "erroneousCallRate":     75,
          "duration":       "10000",
          "httpMethods":       []interface{}{"GET"},
          "hostAddress": "*",
          "urlPath": "/test",
          "failureCauses": []interface{}{"HTTP_502"},
        },
        ExecutionId: uuid.New(),
        Target: extutil.Ptr(action_kit_api.Target{
          Attributes: map[string][]string{
            "process.pid": {"42"},
          },
        }),
      },

      wantedState: &HttpClientStatusState{
        AttackState: &AttackState{
          ConfigJson: "{\"attack-class\":\"com.steadybit.attacks.javaagent.instrumentation.SpringHttpClientStatusInstrumentation\",\"duration\":10000,\"erroneousCallRate\":75,\"failureCauses\":[\"HTTP_502\"],\"hostAddress\":\"*\",\"httpMethods\":[\"GET\"],\"urlPath\":\"/test\"}",
        },
      },
    },
  }
  action := NewHttpClientStatus()
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
