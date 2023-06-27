// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
	"github.com/steadybit/discovery-kit/go/discovery_kit_api"
	"github.com/steadybit/extension-jvm/extjvm"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"strconv"
	"testing"
	"time"
)

var (
	springBootSample       *SpringBootSample
	deleteSpringBootSample func()
	pid                    int32
)

func TestWithMinikube(t *testing.T) {

	extFactory := e2e.HelmExtensionFactory{
		Name: "extension-jvm",
		Port: 8087,
		ExtraArgs: func(m *e2e.Minikube) []string {
			return []string{
				"--set", fmt.Sprintf("container.runtime=%s", m.Runtime),
				"--set", "logging.level=INFO",
			}
		},
		BeforeAllFunc: func(t *testing.T, m *e2e.Minikube, e *e2e.Extension) error {
			springBootSample, pid, deleteSpringBootSample = initTest(t, m, e)
			return nil
		},

		AfterAllFunc: func(t *testing.T, m *e2e.Minikube, e *e2e.Extension) error {
			deleteSpringBootSample()
			return nil
		},
	}

	mOpts := e2e.DefaultMiniKubeOpts
	if os.Getenv("CI") == "true" {
		mOpts.Runtimes = []e2e.Runtime{e2e.RuntimeDocker, e2e.RuntimeContainerd}
	} else {
		mOpts.Runtimes = []e2e.Runtime{e2e.RuntimeDocker}
	}
	//mOpts.Runtimes =e2e.AllRuntimes

	e2e.WithMinikube(t, mOpts, &extFactory, []e2e.WithMinikubeTestCase{
		{
			Name: "discover spring boot sample",
			Test: testDiscoverSpringBootSample,
		},
		{
			Name: "mvc delay",
			Test: testMvcDelay,
		},
		{
			Name: "mvc exception",
			Test: testMvcException,
		},
		{
			Name: "http client delay",
			Test: testHttpClientDelay,
		},
		{
			Name: "http client status",
			Test: testHttpClientStatus,
		},
		{
			Name: "java method delay",
			Test: testJavaMethodDelay,
		},
		{
			Name: "java method exception",
			Test: testJavaMethodException,
		},
		{
			Name: "jdbc template delay",
			Test: testJDBCTemplateDelay,
		},
		{
			Name: "jdbc template exception",
			Test: testJDBCTemplateException,
		},
	})
}

func testDiscoverSpringBootSample(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {
	log.Info().Msg("Starting testDiscoverSpringBootSample")
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	if os.Getenv("CI") == "true" {
		fashionBestseller := FashionBestseller{Minikube: m}
		err := fashionBestseller.Deploy("fashion-bestseller")
		require.NoError(t, err, "failed to create pod")
		defer func() { _ = fashionBestseller.Delete() }()

		go m.TailLog(ctx, fashionBestseller.Pod)
	}

	target := getSpringBootSampleTarget(t, ctx, e)
	assert.Equal(t, target.TargetType, "application")

	if os.Getenv("CI") == "true" {
		targetFashion, err := e2e.PollForTarget(ctx, e, "application", func(target discovery_kit_api.Target) bool {
			log.Debug().Msgf("targetApplications: %+v", target.Attributes)
			return e2e.HasAttribute(target, "application.name", "fashion-bestseller")
		})
		require.NoError(t, err)
		assert.Equal(t, targetFashion.TargetType, "application")
	}
}

func getSpringBootSampleTarget(t *testing.T, ctx context.Context, e *e2e.Extension) discovery_kit_api.Target {
	target, err := e2e.PollForTarget(ctx, e, "application", func(target discovery_kit_api.Target) bool {
		//log.Debug().Msgf("targetApplications: %+v", target.Attributes)
		return e2e.HasAttribute(target, "application.name", "/app") &&
			e2e.HasAttribute(target, "application.type", "spring-boot") &&
			e2e.HasAttribute(target, "spring.application.name", "spring-boot-sample") &&
			e2e.HasAttribute(target, "spring.http-client", "true") &&
			e2e.HasAttribute(target, "datasource.jdbc-url", "jdbc:h2:mem:testdb") &&
			e2e.HasAttribute(target, "spring.jdbc-template", "true")
	})
	require.NoError(t, err)
	return target
}

func testMvcDelay(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
	tests := []struct {
		name          string
		delay         uint64
		jitter        bool
		expectedDelay bool
	}{
		{
			name:          "should not delay traffic",
			expectedDelay: false,
		},
		{
			name:          "should delay traffic",
			delay:         200,
			jitter:        false,
			expectedDelay: true,
		},
		{
			name:          "should delay traffic with jitter",
			delay:         200,
			jitter:        true,
			expectedDelay: true,
		},
	}

	for _, tt := range tests {
		config := struct {
			Duration int    `json:"duration"`
			Delay    uint64 `json:"delay"`
			Jitter   bool   `json:"delayJitter"`
			Pattern  string `json:"pattern"`
			Method   string `json:"method"`
		}{
			Duration: 10000,
			Delay:    tt.delay,
			Jitter:   tt.jitter,
			Pattern:  "/customers",
			Method:   "GET",
		}

		t.Run(tt.name, func(t *testing.T) {
			springBootSample.AssertIsReachable(t, true)

			//measure customer endpoint
			unaffectedLatency, err := springBootSample.MeasureLatency(200)
			require.NoError(t, err, "failed to measure customers endpoint")

			action, err := e.RunAction(extjvm.TargetID+".spring-mvc-delay-attack", &action_kit_api.Target{
				Name: "spring.application.name",
				Attributes: map[string][]string{
					"spring.application.name": {"spring-boot-sample"},
					"process.pid":             {strconv.Itoa(int(pid))},
				},
			}, config, nil)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)
			if tt.expectedDelay {
				springBootSample.AssertLatency(t, getMinLatency(unaffectedLatency, config.Delay), getMaxLatency(unaffectedLatency, config.Delay), unaffectedLatency)
			} else {
				springBootSample.AssertLatency(t, 1*time.Millisecond, unaffectedLatency*2*time.Millisecond, unaffectedLatency)
			}
			require.NoError(t, action.Cancel())
		})
	}
}

func testMvcException(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {
	tests := []struct {
		name              string
		erroneousCallRate int
	}{
		{
			name:              "should not throw exceptions",
			erroneousCallRate: 0,
		},
		{
			name:              "should throw exceptions",
			erroneousCallRate: 100,
		},
	}

	for _, tt := range tests {
		config := struct {
			Duration          int    `json:"duration"`
			ErroneousCallRate int    `json:"erroneousCallRate"`
			Pattern           string `json:"pattern"`
			Method            string `json:"method"`
		}{
			Duration:          10000,
			ErroneousCallRate: tt.erroneousCallRate,
			Pattern:           "/customers",
			Method:            "GET",
		}

		t.Run(tt.name, func(t *testing.T) {
			springBootSample.AssertIsReachable(t, true)

			action, err := e.RunAction(extjvm.TargetID+".spring-mvc-exception-attack", &action_kit_api.Target{
				Name: "spring.application.name",
				Attributes: map[string][]string{
					"spring.application.name": {"spring-boot-sample"},
					"process.pid":             {strconv.Itoa(int(pid))},
				},
			}, config, nil)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)
			if tt.erroneousCallRate > 0 {
				springBootSample.AssertStatus(t, 500)
			} else {
				springBootSample.AssertStatus(t, 200)
			}
			require.NoError(t, action.Cancel())
		})
	}
}

func testHttpClientDelay(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {

	tests := []struct {
		name          string
		delay         uint64
		jitter        bool
		hostAddress   string
		expectedDelay bool
	}{
		{
			name:          "should not delay http client traffic",
			expectedDelay: false,
		},
		{
			name:          "should delay http client traffic",
			delay:         200,
			jitter:        false,
			expectedDelay: true,
		},
		{
			name:          "should delay http client traffic on host",
			delay:         200,
			jitter:        false,
			expectedDelay: true,
			hostAddress:   "www.github.com",
		},
		{
			name:          "should not delay http client traffic on host",
			delay:         200,
			jitter:        false,
			expectedDelay: false,
			hostAddress:   "steadybit.github.com",
		},
		{
			name:          "should delay http client traffic with jitter",
			delay:         200,
			jitter:        true,
			expectedDelay: true,
		},
	}

	for _, tt := range tests {

		if tt.hostAddress == "" {
			tt.hostAddress = "*"
		}

		config := struct {
			Duration    int    `json:"duration"`
			Delay       uint64 `json:"delay"`
			Jitter      bool   `json:"delayJitter"`
			HostAddress string `json:"hostAddress"`
		}{
			Duration:    10000,
			Delay:       tt.delay,
			Jitter:      tt.jitter,
			HostAddress: tt.hostAddress,
		}

		t.Run(tt.name, func(t *testing.T) {
			springBootSample.AssertIsReachable(t, true)

			//measure customer endpoint
			unaffectedLatency, err := springBootSample.MeasureLatencyOnPath(200, "/remote/blocking?url=https://www.github.com")
			require.NoError(t, err, "failed to measure customers endpoint")

			action, err := e.RunAction(extjvm.TargetID+".spring-httpclient-delay-attack", &action_kit_api.Target{
				Name: "spring.application.name",
				Attributes: map[string][]string{
					"spring.application.name": {"spring-boot-sample"},
					"process.pid":             {strconv.Itoa(int(pid))},
				},
			}, config, nil)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)
			if tt.expectedDelay {
				springBootSample.AssertLatencyOnPath(t, getMinLatency(unaffectedLatency, config.Delay), getMaxLatency(unaffectedLatency, config.Delay), "/remote/blocking?url=https://www.github.com", unaffectedLatency)
			} else {
				springBootSample.AssertLatencyOnPath(t, 1*time.Millisecond, unaffectedLatency*2*time.Millisecond, "/remote/blocking?url=https://www.github.com", 0)
			}
			require.NoError(t, action.Cancel())
		})
	}
}

func getMinLatency(unaffectedLatency time.Duration, delay uint64) time.Duration {
	return unaffectedLatency + time.Duration(delay)*time.Millisecond*60/100
}

func getMaxLatency(unaffectedLatency time.Duration, delay uint64) time.Duration {
	return unaffectedLatency + time.Duration(delay)*time.Millisecond*350/100
}

func testHttpClientStatus(t *testing.T, m *e2e.Minikube, e *e2e.Extension) {

	tests := []struct {
		name               string
		erroneousCallRate  int
		hostAddress        string
		expectedLogStatus  int
		expectedHttpStatus int
		failureTypes       []string
	}{
		{
			name:               "should not throw http client exceptions",
			erroneousCallRate:  0,
			expectedHttpStatus: 200,
			failureTypes:       []string{},
		},
		{
			name:               "should throw http client exceptions",
			erroneousCallRate:  100,
			failureTypes:       []string{},
			expectedLogStatus:  500,
			expectedHttpStatus: 500,
		},
		{
			name:               "should throw http client exceptions on host and set http status",
			erroneousCallRate:  100,
			failureTypes:       []string{"HTTP_502"},
			expectedLogStatus:  502,
			expectedHttpStatus: 500,
			hostAddress:        "www.github.com",
		},
		{
			name:               "should not throw http client exceptions on host",
			erroneousCallRate:  100,
			failureTypes:       []string{},
			expectedLogStatus:  200,
			expectedHttpStatus: 200,
			hostAddress:        "steadybit.github.com",
		},
		{
			name:               "should throw http client exceptions sometimes",
			erroneousCallRate:  50,
			failureTypes:       []string{},
			expectedLogStatus:  500,
			expectedHttpStatus: 500,
		},
	}

	for _, tt := range tests {
		if tt.hostAddress == "" {
			tt.hostAddress = "*"
		}

		config := struct {
			Duration          int      `json:"duration"`
			ErroneousCallRate int      `json:"erroneousCallRate"`
			HttpMethods       []string `json:"httpMethods"`
			HostAddress       string   `json:"hostAddress"`
			UrlPath           string   `json:"urlPath"`
			FailureCauses     []string `json:"failureCauses"`
		}{
			Duration:          10000,
			ErroneousCallRate: tt.erroneousCallRate,
			HttpMethods:       []string{"GET"},
			HostAddress:       tt.hostAddress,
			UrlPath:           "*",
			FailureCauses:     tt.failureTypes,
		}

		t.Run(tt.name, func(t *testing.T) {
			springBootSample.AssertIsReachable(t, true)

			action, err := e.RunAction(extjvm.TargetID+".spring-httpclient-status-attack", &action_kit_api.Target{
				Name: "spring.application.name",
				Attributes: map[string][]string{
					"spring.application.name": {"spring-boot-sample"},
					"process.pid":             {strconv.Itoa(int(pid))},
				},
			}, config, nil)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)
			if tt.erroneousCallRate > 0 {
				springBootSample.AssertStatusOnPath(t, tt.expectedHttpStatus, "/remote/blocking?url=https://www.github.com")
				if tt.expectedLogStatus != 200 {
					e2e.AssertLogContains(t, m, springBootSample.Pod, strconv.Itoa(tt.expectedLogStatus)+" Injected by steadybit")
				}
			} else {
				springBootSample.AssertStatusOnPath(t, tt.expectedHttpStatus, "/remote/blocking?url=https://www.github.com")
			}
			require.NoError(t, action.Cancel())
		})
	}
}

func testJavaMethodDelay(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {

	tests := []struct {
		name          string
		delay         uint64
		jitter        bool
		expectedDelay bool
	}{
		{
			name:          "should not delay java method execution",
			expectedDelay: false,
		},
		{
			name:          "should delay java method execution",
			delay:         200,
			jitter:        false,
			expectedDelay: true,
		},
		{
			name:          "should delay java method execution with jitter",
			delay:         200,
			jitter:        true,
			expectedDelay: true,
		},
	}

	for _, tt := range tests {

		config := struct {
			Duration   int    `json:"duration"`
			Delay      uint64 `json:"delay"`
			Jitter     bool   `json:"delayJitter"`
			ClassName  string `json:"className"`
			MethodName string `json:"methodName"`
		}{
			Duration:   10000,
			Delay:      tt.delay,
			Jitter:     tt.jitter,
			ClassName:  "com.steadybit.samples.data.CustomerController",
			MethodName: "getAllCustomers",
		}

		t.Run(tt.name, func(t *testing.T) {
			springBootSample.AssertIsReachable(t, true)

			//measure customer endpoint
			unaffectedLatency, err := springBootSample.MeasureLatency(200)
			require.NoError(t, err, "failed to measure customers endpoint")

			action, err := e.RunAction(extjvm.TargetID+".java-method-delay-attack", &action_kit_api.Target{
				Name: "spring.application.name",
				Attributes: map[string][]string{
					"spring.application.name": {"spring-boot-sample"},
					"process.pid":             {strconv.Itoa(int(pid))},
				},
			}, config, nil)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)
			if tt.expectedDelay {
				springBootSample.AssertLatency(t, getMinLatency(unaffectedLatency, config.Delay), getMaxLatency(unaffectedLatency, config.Delay), unaffectedLatency)
			} else {
				springBootSample.AssertLatency(t, 1*time.Millisecond, unaffectedLatency*2*time.Millisecond, unaffectedLatency)
			}
			require.NoError(t, action.Cancel())
		})
	}
}

func testJavaMethodException(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {

	tests := []struct {
		name              string
		erroneousCallRate int
	}{
		{
			name:              "should not throw exceptions",
			erroneousCallRate: 0,
		},
		{
			name:              "should throw exceptions",
			erroneousCallRate: 100,
		},
	}

	for _, tt := range tests {
		config := struct {
			Duration          int    `json:"duration"`
			ErroneousCallRate int    `json:"erroneousCallRate"`
			ClassName         string `json:"className"`
			MethodName        string `json:"methodName"`
		}{
			Duration:          10000,
			ErroneousCallRate: tt.erroneousCallRate,
			ClassName:         "com.steadybit.samples.data.CustomerController",
			MethodName:        "getAllCustomers",
		}

		t.Run(tt.name, func(t *testing.T) {
			springBootSample.AssertIsReachable(t, true)

			action, err := e.RunAction(extjvm.TargetID+".java-method-exception-attack", &action_kit_api.Target{
				Name: "spring.application.name",
				Attributes: map[string][]string{
					"spring.application.name": {"spring-boot-sample"},
					"process.pid":             {strconv.Itoa(int(pid))},
				},
			}, config, nil)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)
			if tt.erroneousCallRate > 0 {
				springBootSample.AssertStatus(t, 500)
			} else {
				springBootSample.AssertStatus(t, 200)
			}
			require.NoError(t, action.Cancel())
		})
	}
}

func testJDBCTemplateDelay(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {

	tests := []struct {
		name          string
		delay         uint64
		jitter        bool
		expectedDelay bool
		operations    string
		jdbcUrl       string
	}{
		{
			name:          "should not delay jdbc template execution",
			expectedDelay: false,
		},
		{
			name:          "should delay jdbc template execution",
			delay:         200,
			jitter:        false,
			expectedDelay: true,
		},
		{
			name:          "should delay jdbc template execution with jitter",
			delay:         200,
			jitter:        true,
			expectedDelay: true,
		},
		{
			name:          "should delay jdbc template execution with jitter only for reads",
			delay:         200,
			jitter:        true,
			expectedDelay: true,
			operations:    "r",
		},
		{
			name:          "should not delay jdbc template execution with jitter only for writes",
			delay:         200,
			jitter:        true,
			expectedDelay: false,
			operations:    "w",
		},
		{
			name:          "should not delay jdbc template execution on unknown url",
			delay:         200,
			jitter:        true,
			expectedDelay: false,
			operations:    "*",
			jdbcUrl:       "jdbc:gibtesnicht",
		},
		{
			name:          "should delay jdbc template execution with jitter on jdbc:h2:mem:testdb",
			delay:         200,
			jitter:        true,
			expectedDelay: true,
			operations:    "*",
			jdbcUrl:       "jdbc:h2:mem:testdb",
		},
	}

	for _, tt := range tests {
		if tt.jdbcUrl == "" {
			tt.jdbcUrl = "*"
		}
		if tt.operations == "" {
			tt.operations = "*"
		}

		config := struct {
			Duration   int    `json:"duration"`
			Delay      uint64 `json:"delay"`
			Jitter     bool   `json:"delayJitter"`
			JdbcUrl    string `json:"jdbcUrl"`
			Operations string `json:"operations"`
		}{
			Duration:   10000,
			Delay:      tt.delay,
			Jitter:     tt.jitter,
			JdbcUrl:    tt.jdbcUrl,
			Operations: tt.operations,
		}

		t.Run(tt.name, func(t *testing.T) {
			springBootSample.AssertIsReachable(t, true)

			//measure customer endpoint
			unaffectedLatency, err := springBootSample.MeasureLatency(200)
			require.NoError(t, err, "failed to measure customers endpoint")

			action, err := e.RunAction(extjvm.TargetID+".spring-jdbctemplate-delay-attack", &action_kit_api.Target{
				Name: "spring.application.name",
				Attributes: map[string][]string{
					"spring.application.name": {"spring-boot-sample"},
					"process.pid":             {strconv.Itoa(int(pid))},
				},
			}, config, nil)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)
			if tt.expectedDelay {
				springBootSample.AssertLatency(t, getMinLatency(unaffectedLatency, config.Delay), getMaxLatency(unaffectedLatency, config.Delay), unaffectedLatency)
			} else {
				springBootSample.AssertLatency(t, 1*time.Millisecond, unaffectedLatency*2*time.Millisecond, unaffectedLatency)
			}
			require.NoError(t, action.Cancel())
		})
	}
}

func testJDBCTemplateException(t *testing.T, _ *e2e.Minikube, e *e2e.Extension) {

	tests := []struct {
		name              string
		erroneousCallRate int
		operations        string
		jdbcUrl           string
	}{
		{
			name:              "should not throw exceptions",
			erroneousCallRate: 0,
		},
		{
			name:              "should throw exceptions",
			erroneousCallRate: 100,
		},
		{
			name:              "should throw exceptions with jdbc:h2:mem:testdb",
			erroneousCallRate: 90,
			operations:        "*",
			jdbcUrl:           "jdbc:h2:mem:testdb",
		},
	}

	for _, tt := range tests {
		if tt.jdbcUrl == "" {
			tt.jdbcUrl = "*"
		}
		if tt.operations == "" {
			tt.operations = "*"
		}
		config := struct {
			Duration          int    `json:"duration"`
			ErroneousCallRate int    `json:"erroneousCallRate"`
			ClassName         string `json:"className"`
			MethodName        string `json:"methodName"`
		}{
			Duration:          10000,
			ErroneousCallRate: tt.erroneousCallRate,
			ClassName:         "com.steadybit.samples.data.CustomerController",
			MethodName:        "getAllCustomers",
		}

		t.Run(tt.name, func(t *testing.T) {
			springBootSample.AssertIsReachable(t, true)

			action, err := e.RunAction(extjvm.TargetID+".java-method-exception-attack", &action_kit_api.Target{
				Name: "spring.application.name",
				Attributes: map[string][]string{
					"spring.application.name": {"spring-boot-sample"},
					"process.pid":             {strconv.Itoa(int(pid))},
				},
			}, config, nil)
			defer func() { _ = action.Cancel() }()
			require.NoError(t, err)
			if tt.erroneousCallRate > 0 {
				springBootSample.AssertStatus(t, 500)
			} else {
				springBootSample.AssertStatus(t, 200)
			}
			require.NoError(t, action.Cancel())
		})
	}
}

func initTest(t *testing.T, m *e2e.Minikube, e *e2e.Extension) (*SpringBootSample, int32, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	sbsApp := deploySpringBootSample(t, m)
	sbsApp.AssertIsReachable(t, true)

	//go m.TailLog(ctx, springBootSample.Pod)

	target := getSpringBootSampleTarget(t, ctx, e)
	p := extutil.ToInt32(target.Attributes["process.pid"][0])
	return extutil.Ptr(sbsApp), p, func() { _ = sbsApp.Delete() }
}

func deploySpringBootSample(t *testing.T, m *e2e.Minikube) SpringBootSample {
	springBootSample := SpringBootSample{Minikube: m}
	err := springBootSample.Deploy("spring-boot-sample")
	require.NoError(t, err, "failed to create pod")
	return springBootSample
}
