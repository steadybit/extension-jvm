package jvmhttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_handleInternal(t *testing.T) {
	type args struct {
		remoteAddress string
		body          string
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "Test 1",
			args: args{
				remoteAddress: "10.244.0.3:42776",
				body:          "3681=33773",
			},
			want: http.StatusOK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			handleInternal(w, tt.args.remoteAddress, tt.args.body)
			got := w.Result().StatusCode
			if got != tt.want {
				t.Errorf("handleInternal() got = %v, want %v", got, tt.want)
			}
		})
	}
}
