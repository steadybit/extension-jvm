// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
  "context"
  "github.com/rs/zerolog/log"
  "github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
  "github.com/steadybit/discovery-kit/go/discovery_kit_api"
  "github.com/stretchr/testify/assert"
  "github.com/stretchr/testify/require"
	"testing"
  "time"
)

func TestWithMinikube(t *testing.T) {
	extFactory := e2e.HelmExtensionFactory{
		Name: "extension-jvm",
		Port: 8085,
		ExtraArgs: func(m *e2e.Minikube) []string {
			return []string{"--set", "logging.level=INFO"}
		},
	}

	mOpts := e2e.DefaultMiniKubeOpts
	mOpts.Runtimes = []e2e.Runtime{e2e.RuntimeDocker}

	e2e.WithMinikube(t, mOpts, &extFactory, []e2e.WithMinikubeTestCase{
		//{
		//	Name: "run jvm",
		//	Test: testRunJVM,
		//},
    {
			Name: "discover spring boot sample",
			Test: testDiscoverSpringBootSample,
		},
	})
}

func testDiscoverSpringBootSample(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
  log.Info().Msg("Starting testDiscoverSpringBootSample")
  ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
  defer cancel()

  springBootSample := SpringBootSample{Minikube: m}
  err := springBootSample.Deploy("spring-boot-sample")
  require.NoError(t, err, "failed to create pod")
  defer func() { _ = springBootSample.Delete() }()


  target, err := e2e.PollForTarget(ctx, e, "application", func(target discovery_kit_api.Target) bool {
    log.Debug().Msgf("targetApplications: %+v", target.Attributes)
    return e2e.HasAttribute(target, "application.name", "/app") && e2e.HasAttribute(target, "application.type", "spring-boot")
  })

  require.NoError(t, err)
  assert.Equal(t, target.TargetType, "application")
}

func testRunJVM(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	config := struct{}{}
	exec, err := e.RunAction("application.log", &action_kit_api.Target{
		Name: "robot",
	}, config, nil)
	require.NoError(t, err)
	e2e.AssertLogContains(t, m, e.Pod, "Logging in log action **start**")
	require.NoError(t, exec.Cancel())
}
