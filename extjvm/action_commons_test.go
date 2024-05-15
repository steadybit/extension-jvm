package extjvm

import (
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"os/exec"
	"strconv"
)

type fakeJvm struct {
	cmd *exec.Cmd
}

func startFakeJvm() (fakeJvm, error) {
	cmd := exec.Command("sleep", "120")
	if err := cmd.Start(); err != nil {
		return fakeJvm{}, err
	}

	addJvm(&jvm.JavaVm{
		Pid: int32(cmd.Process.Pid),
	})

	springApplications.Store(cmd.Process.Pid, SpringApplication{
		Name: "customers",
		Pid:  int32(cmd.Process.Pid),
		MvcMappings: &[]SpringMvcMapping{
			{
				Methods:      []string{"GET"},
				Patterns:     []string{"/customers"},
				HandlerClass: "com.steadybit.demo.CustomerController",
				HandlerName:  "customers",
			},
		},
	})

	return fakeJvm{cmd}, nil
}

func (f *fakeJvm) getTarget() action_kit_api.Target {
	return action_kit_api.Target{
		Attributes: map[string][]string{
			"process.pid": {strconv.Itoa(f.cmd.Process.Pid)},
		},
	}
}

func (f *fakeJvm) stop() error {
	return f.cmd.Process.Kill()
}
