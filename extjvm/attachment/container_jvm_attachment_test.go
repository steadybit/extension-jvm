package attachment

import (
  "testing"
)

func Test_getOutboundIP(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
      name: "Test_getOutboundIP",
    },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getOutboundIP(); got == nil {
				t.Errorf("getOutboundIP() = %v, want %v", got, "not nil")
			}
		})
	}
}
