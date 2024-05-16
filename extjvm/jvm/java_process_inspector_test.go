package jvm

import (
	"github.com/steadybit/extension-jvm/extjvm/jvm/hsperf"
	"github.com/steadybit/extension-jvm/extjvm/jvm/test"
	"github.com/stretchr/testify/assert"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

var hostname string

func init() {
	hostname, _ = os.Hostname()
}

func Test_should_inspect_host_process_via_process(t *testing.T) {
	jvm := test.NewSleep()
	defer jvm.Stop()

	inspector := JavaProcessInspector{}
	inspector.Start()
	defer inspector.Stop()

	//the test is usual faster than the hsperfdata files appear and therefore uses the process to inspect
	inspector.Inspect(jvm.Process(), 0, "test")

	hostname, _ := os.Hostname()

	select {
	case j := <-inspector.JavaVms:
		assert.Equal(t, jvm.Pid(), j.Pid())
		assert.Contains(t, j.CommandLine(), "60000")
		assert.Equal(t, strconv.Itoa(os.Geteuid()), j.UserId())
		assert.Equal(t, strconv.Itoa(os.Geteuid()), j.UserId())
		assert.Equal(t, hostname, j.Hostname())
		assert.True(t, j.IsRunning())
		assert.Condition(t, func() bool {
			return strings.HasSuffix(j.Path(), "/bin/java")
		})
		assert.Condition(t, func() bool {
			return strings.HasSuffix(j.(*defaultJavaVm).discoveredVia, "os-process")
		})

		jvm.Stop()
		assert.False(t, j.IsRunning())

	case <-time.After(5 * time.Second):
		assert.Fail(t, "jvm not inspected")
	}
}

func Test_should_inspect_host_process_via_hsperf(t *testing.T) {
	jvm := test.NewSleep()
	defer jvm.Stop()

	inspector := JavaProcessInspector{}
	inspector.Start()
	defer inspector.Stop()

	w := hsperf.Watcher{}
	w.Start()
	defer w.Stop()

	//We wait for the JVM to be discovered via hsperfdata
	p := test.RequireProcessEmitted(t, w.Processes, jvm.Pid())
	drain(w.Processes)

	inspector.Inspect(p, 5, "test")

	select {
	case j := <-inspector.JavaVms:
		assert.Equal(t, jvm.Pid(), j.Pid())
		assert.Contains(t, j.CommandLine(), "60000")
		assert.Equal(t, strconv.Itoa(os.Geteuid()), j.UserId())
		assert.Equal(t, strconv.Itoa(os.Geteuid()), j.UserId())
		assert.Equal(t, hostname, j.Hostname())
		assert.True(t, j.IsRunning())
		assert.Condition(t, func() bool {
			return strings.HasSuffix(j.Path(), "/bin/java")
		})
		assert.Condition(t, func() bool {
			return strings.HasSuffix(j.(*defaultJavaVm).discoveredVia, "hsperfdata")
		})
		assert.Equal(t, "Main", j.MainClass())
		assert.NotEmpty(t, ".", j.ClassPath())

		jvm.Stop()
		assert.False(t, j.IsRunning())

	case <-time.After(5 * time.Second):
		assert.Fail(t, "jvm not inspected")
	}

}

func drain[T any](ch <-chan T) {
	for {
		select {
		case <-ch:
		case <-time.After(10 * time.Millisecond):
			return
		}
	}
}
