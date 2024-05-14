package utils

import (
	"context"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"syscall"
)

func ContainsPartOfString(s []string, str string) bool {
	return slices.ContainsFunc(s, func(v string) bool {
		return strings.Contains(str, v)
	})
}

var RootCommandContextDisableSetuid = false

func RootCommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, arg...)
	if runtime.GOOS != "darwin" && !RootCommandContextDisableSetuid {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid: 0,
				Gid: 0,
			},
		}
	}
	return cmd
}

func AppendIfMissing(slice []string, val string) []string {
	for _, ele := range slice {
		if ele == val {
			return slice
		}
	}
	return append(slice, val)
}
