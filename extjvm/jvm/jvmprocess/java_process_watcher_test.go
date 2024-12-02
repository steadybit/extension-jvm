// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2024 Steadybit GmbH

package jvmprocess

import (
	"github.com/steadybit/extension-jvm/extjvm/jvm/test"
	"github.com/steadybit/extension-jvm/extjvm/utils"
	"testing"
	"time"
)

func Test_should_find_java_application(t *testing.T) {
	utils.RootCommandContextDisableSetuid = true
	defer func() { utils.RootCommandContextDisableSetuid = false }()

	existingJvm := test.NewSleep()
	defer existingJvm.Stop()

	w := ProcessWatcher{Interval: 1 * time.Second}
	w.Start()
	defer w.Stop()

	test.AssertProcessEmitted(t, w.Processes, existingJvm.Pid())

	newJvm := test.NewSleep()
	defer newJvm.Stop()

	test.AssertProcessEmitted(t, w.Processes, newJvm.Pid())
}
