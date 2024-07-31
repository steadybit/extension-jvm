package internal

import (
	"github.com/steadybit/extension-jvm/extjvm/jvm/test"
	"testing"
	"time"
)

func Test_should_find_java_application(t *testing.T) {
	existingJvm := test.NewSleep()
	defer existingJvm.Stop()

	w := ProcessWatcher{}
	w.StartWithInterval(1 * time.Second)
	defer w.Stop()

	test.AssertProcessEmitted(t, w.Processes, existingJvm.Pid())

	newJvm := test.NewSleep()
	defer newJvm.Stop()

	test.AssertProcessEmitted(t, w.Processes, newJvm.Pid())
}
