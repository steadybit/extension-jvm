package java_process

import (
	"testing"
)

func Test_removePidFromDiscoveredPids(t *testing.T) {
	type args struct {
		pid int32
	}
	tests := []struct {
		name string
		args args
	}{
		{"Test_removePidFromDiscoveredPids", args{pid: 42}},
	}
	for _, tt := range tests {
		discoveredPids = append(discoveredPids, 1,2,3,4)
		discoveredPids = append(discoveredPids, tt.args.pid)
		discoveredPids = append(discoveredPids, 5,6,7)
		t.Run(tt.name, func(t *testing.T) {
			RemovePidFromDiscoveredPids(tt.args.pid)
			for _, p := range discoveredPids {
				if p == tt.args.pid {
					t.Errorf("removePidFromDiscoveredPids() did not remove pid %d", tt.args.pid)
				}
			}
		})
	}
}
