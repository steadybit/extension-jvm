// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package hsperf

import (
	"github.com/steadybit/extension-jvm/extjvm/jvm/test"
	"testing"
)

func Test_should_find_java_application(t *testing.T) {
	existingJvm := test.NewSleep()
	defer existingJvm.Stop()

	w := Watcher{}
	w.Start()
	defer w.Stop()

	test.AssertProcessEmitted(t, w.Processes, existingJvm.Pid())

	newJvm := test.NewSleep()
	defer newJvm.Stop()

	test.AssertProcessEmitted(t, w.Processes, newJvm.Pid())
}
