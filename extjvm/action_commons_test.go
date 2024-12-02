package extjvm

import (
	"fmt"
	"github.com/shirou/gopsutil/v4/process"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/extension-jvm/extjvm/jvm"
	"github.com/stretchr/testify/mock"
	"io"
	"os/exec"
	"strconv"
	"time"
)

type mockJavaFacade struct {
	mock.Mock
	fakes []*FakeJvm
}

func (m *mockJavaFacade) Start() {
	m.Called()
}

func (m *mockJavaFacade) Stop() {
	m.Called()
}

func (m *mockJavaFacade) AddAttachedListener(attachedListener jvm.AttachListener) {
	m.Called(attachedListener)
}

func (m *mockJavaFacade) RemoveAttachedListener(AttachedListener jvm.AttachListener) {
	m.Called(AttachedListener)
}

func (m *mockJavaFacade) LoadAgentPlugin(javaVm jvm.JavaVm, plugin string, args string) error {
	return m.Called(javaVm, plugin, args).Error(0)
}

func (m *mockJavaFacade) UnloadAgentPlugin(javaVm jvm.JavaVm, plugin string) error {
	return m.Called(javaVm, plugin).Error(0)
}

func (m *mockJavaFacade) HasAgentPlugin(javaVm jvm.JavaVm, plugin string) bool {
	return m.Called(javaVm, plugin).Bool(0)
}

func (m *mockJavaFacade) HasClassLoaded(javaVm jvm.JavaVm, className string) bool {
	return m.Called(javaVm, className).Bool(0)
}

func (m *mockJavaFacade) SendCommandToAgent(javaVm jvm.JavaVm, command string, args string) (bool, error) {
	a := m.Called(javaVm, command, args)
	return a.Bool(0), a.Error(1)
}

func (m *mockJavaFacade) SendCommandToAgentWithHandler(javaVm jvm.JavaVm, command string, args string, handler func(response io.Reader) (any, error)) (any, error) {
	a := m.Called(javaVm, command, args, handler)
	return a.Get(0), a.Error(1)
}

func (m *mockJavaFacade) SendCommandToAgentWithTimeout(javaVm jvm.JavaVm, command string, args string, timeout time.Duration) (bool, error) {
	a := m.Called(javaVm, command, args, timeout)
	return a.Bool(0), a.Error(1)
}

func (m *mockJavaFacade) AddAutoloadAgentPlugin(plugin string, markerClass string) {
	m.Called(plugin, markerClass)
}

func (m *mockJavaFacade) RemoveAutoloadAgentPlugin(plugin string, markerClass string) {
	m.Called(plugin, markerClass)
}

func (m *mockJavaFacade) GetJvm(pid int32) jvm.JavaVm {
	for _, fake := range m.fakes {
		if fake.Pid() == pid {
			return fake
		}
	}
	return nil
}

func (m *mockJavaFacade) GetJvms() []jvm.JavaVm {
	c := make([]jvm.JavaVm, len(m.fakes))
	for i, f := range m.fakes {
		c[i] = f
	}
	return c
}

func (m *mockJavaFacade) startFakeJvm() (*FakeJvm, error) {
	cmd := exec.Command("sleep", "120")
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	p, err := process.NewProcess(int32(cmd.Process.Pid))
	if err != nil {
		return nil, err
	}

	fake := &FakeJvm{p: p}
	m.fakes = append(m.fakes, fake)
	return fake, nil
}

type FakeJvm struct {
	p *process.Process
}

func (f *FakeJvm) CreateTime() time.Time {
	t, _ := f.p.CreateTime()
	return time.UnixMilli(t)
}

func (f *FakeJvm) IsRunning() bool {
	b, _ := f.p.IsRunning()
	return b
}

func (f *FakeJvm) Pid() int32 {
	return int32(f.p.Pid)
}

func (f *FakeJvm) CommandLine() string {
	s, _ := f.p.Cmdline()
	return s
}

func (f *FakeJvm) MainClass() string {
	return "fake"
}

func (f *FakeJvm) ClassPath() string {
	return ""
}

func (f *FakeJvm) VmVersion() string {
	return "0"
}

func (f *FakeJvm) VmVendor() string {
	return "fake inc."
}

func (f *FakeJvm) VmName() string {
	return "fake"
}

func (f *FakeJvm) VmArgs() string {
	return ""
}

func (f *FakeJvm) UserId() string {
	return ""
}

func (f *FakeJvm) GroupId() string {
	return ""
}

func (f *FakeJvm) Path() string {
	return ""
}

func (f *FakeJvm) DiscoveredVia() string {
	return "fake"
}

func (f *FakeJvm) Hostname() string {
	return ""
}

func (f *FakeJvm) HostFQDN() string {
	return ""
}

func (f *FakeJvm) ToInfoString() string {
	return fmt.Sprintf("Fake %d", f.Pid())
}

func (f *FakeJvm) ToDebugString() string {
	return fmt.Sprintf("Fake %d", f.Pid())
}

func (f *FakeJvm) getTarget() action_kit_api.Target {
	return action_kit_api.Target{
		Attributes: map[string][]string{
			"process.pid": {strconv.Itoa(int(f.p.Pid))},
		},
	}
}

func (f *FakeJvm) stop() error {
	return f.p.Kill()
}
