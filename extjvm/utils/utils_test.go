package utils

import (
	"reflect"
	"testing"
)

func TestAppendIfMissing(t *testing.T) {
	type args struct {
		slice []string
		val   string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
      name: "Append because it is missing",
      args: args{
        slice: []string{"a", "b", "c"},
        val: "d",
      },
      want: []string{"a", "b", "c", "d"},
    },
    {
      name: "Do not append because it is not missing",
      args: args{
        slice: []string{"a", "b", "c"},
        val: "b",
      },
      want: []string{"a", "b", "c"},
    },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AppendIfMissing(tt.args.slice, tt.args.val); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("AppendIfMissing() = %v, want %v", got, tt.want)
			}
		})
	}
}
