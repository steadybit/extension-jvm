package controller

import "testing"

func Test_handleInternal(t *testing.T) {
	type args struct {
		remoteAddress string
		body          string
	}
	tests := []struct {
		name    string
		args    args
		want    uint16
		wantErr bool
	}{
		{
			name: "Test 1",
			args: args{
				remoteAddress: "10.244.0.3:42776",
				body:          "3681=33773",
			},
			want:    200,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := handleInternal(tt.args.remoteAddress, tt.args.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("handleInternal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("handleInternal() got = %v, want %v", got, tt.want)
			}
		})
	}
}
