package jvm

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_javaagentHttpServer_doJavaagent(t *testing.T) {
	srv := &javaagentHttpServer{
		connections: &jvmConnections{},
	}

	tests := []struct {
		remoteAddress   string
		body            string
		wantErr         string
		wantPid         int32
		wantConnAddress string
	}{
		{
			remoteAddress: "10.244.0.3:42776",
			body:          "",
			wantErr:       "invalid body",
		},
		{
			remoteAddress: "10.244.0.3:42776",
			body:          "A=1234",
			wantErr:       "invalid pid",
		},
		{
			remoteAddress: "10.244.0.3:42776",
			body:          "1234=A",
			wantErr:       "invalid port",
		},
		{
			remoteAddress: "10.244.0.3:42776",
			body:          "1234=this.is.not.valid:80",
			wantErr:       "unknown host",
		},
		{
			remoteAddress: "10.244.0.3:42776",
			body:          "1234=8.8.8.8:A",
			wantErr:       "invalid port",
		},
		{
			remoteAddress:   "10.244.0.3:42776",
			body:            "1234=1111",
			wantPid:         1234,
			wantConnAddress: "10.244.0.3:1111",
		},
		{
			remoteAddress:   "10.244.0.3:42776",
			body:            "1234=8.8.8.8:3333",
			wantPid:         1234,
			wantConnAddress: "8.8.8.8:3333",
		},
	}

	for _, tt := range tests {
		name := fmt.Sprintf("doJavaagent(%s %s", tt.remoteAddress, tt.body)
		t.Run(name, func(t *testing.T) {
			err := srv.doJavaagent(tt.remoteAddress, tt.body)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
			if tt.wantConnAddress != "" {
				conn := srv.connections.getConnection(tt.wantPid)
				assert.Equal(t, conn.Address, tt.wantConnAddress)
			}
		})
	}
}
