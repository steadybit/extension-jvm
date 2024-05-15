package utils

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
)

func Contains(s []int32, str int32) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func ContainsString(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

func ContainsPartOfString(s []string, str string) bool {
	for _, v := range s {
		if strings.Contains(str, v) {
			return true
		}
	}
	return false
}

func RootCommandContext(ctx context.Context, name string, arg ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, arg...)
	if runtime.GOOS != "darwin" {
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
