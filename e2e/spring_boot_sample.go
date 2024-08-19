// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"errors"
	"fmt"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
	"github.com/steadybit/extension-kit/extutil"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	acorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	ametav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"testing"
	"time"
)

type SpringBootSample struct {
	Minikube *e2e.Minikube
	Pod      metav1.Object
	Service  metav1.Object
}

func (n *SpringBootSample) Deploy(podName string, opts ...func(c *acorev1.PodApplyConfiguration)) error {
	cfg := &acorev1.PodApplyConfiguration{
		TypeMetaApplyConfiguration: ametav1.TypeMetaApplyConfiguration{
			Kind:       extutil.Ptr("Pod"),
			APIVersion: extutil.Ptr("v1"),
		},
		ObjectMetaApplyConfiguration: &ametav1.ObjectMetaApplyConfiguration{
			Name:   &podName,
			Labels: map[string]string{"app": podName},
		},
		Spec: &acorev1.PodSpecApplyConfiguration{
			RestartPolicy: extutil.Ptr(corev1.RestartPolicyNever),
			Containers: []acorev1.ContainerApplyConfiguration{
				{
					Name:  extutil.Ptr("spring-boot-sample"),
					Image: extutil.Ptr("docker.io/steadybit/spring-boot-sample:1.0.24"),
					Ports: []acorev1.ContainerPortApplyConfiguration{
						{
							ContainerPort: extutil.Ptr(int32(80)),
						},
					},
					Env: []acorev1.EnvVarApplyConfiguration{
						{
							Name:  extutil.Ptr("STEADYBIT_LOG_JAVAAGENT_STDOUT"),
							Value: extutil.Ptr("true"),
						},
						{
							Name:  extutil.Ptr("STEADYBIT_LOG_LEVEL"),
							Value: extutil.Ptr("DEBUG"),
						},
					},
				},
			},
		},
		Status: nil,
	}

	for _, fn := range opts {
		fn(cfg)
	}

	pod, err := n.Minikube.CreatePod(cfg)
	if err != nil {
		return err
	}
	n.Pod = pod

	service, err := n.Minikube.CreateService(&acorev1.ServiceApplyConfiguration{
		TypeMetaApplyConfiguration: ametav1.TypeMetaApplyConfiguration{
			Kind:       extutil.Ptr("Service"),
			APIVersion: extutil.Ptr("v1"),
		},
		ObjectMetaApplyConfiguration: &ametav1.ObjectMetaApplyConfiguration{
			Name:   extutil.Ptr("spring-boot-sample"),
			Labels: map[string]string{"app": podName},
		},
		Spec: &acorev1.ServiceSpecApplyConfiguration{
			Type:     extutil.Ptr(corev1.ServiceTypeLoadBalancer),
			Selector: n.Pod.GetLabels(),
			Ports: []acorev1.ServicePortApplyConfiguration{
				{
					Port:     extutil.Ptr(int32(8080)),
					Protocol: extutil.Ptr(corev1.ProtocolTCP),
				},
			},
		},
	})
	if err != nil {
		return err
	}
	n.Service = service

	return nil
}

func (n *SpringBootSample) Target() (*action_kit_api.Target, error) {
	return e2e.NewContainerTarget(n.Minikube, n.Pod, "spring-boot-sample")
}

func (n *SpringBootSample) IsReachable() error {
	client, err := n.Minikube.NewRestClientForService(n.Service)
	if err != nil {
		return err
	}
	defer client.Close()

	_, err = client.R().Get("/customers")
	if err != nil {
		return err
	}

	return nil
}

func (n *SpringBootSample) AssertIsReachable(t *testing.T, expected bool) {
	t.Helper()

	client, err := n.Minikube.NewRestClientForService(n.Service)
	require.NoError(t, err)
	defer client.Close()

	e2e.Retry(t, 20, 500*time.Millisecond, func(r *e2e.R) {
		_, err = client.R().Get("/customers")
		if expected && err != nil {
			r.Failed = true
			_, _ = fmt.Fprintf(r.Log, "expected spring boot sample to be reachble, but was not: %s", err)
		} else if !expected && err == nil {
			r.Failed = true
			_, _ = fmt.Fprintf(r.Log, "expected spring boot sample not to be reachble, but was")
		}
	})
}

func (n *SpringBootSample) GetHttpStatusOfCustomersEndpoint() (int, error) {
	client, err := n.Minikube.NewRestClientForService(n.Service)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	response, err := client.R().Get("/customers")
	if err != nil {
		return 0, err
	}

	return response.StatusCode(), nil
}

func (n *SpringBootSample) MeasureLatency(expectedStatus int) (time.Duration, error) {
	return n.MeasureLatencyOnPath(expectedStatus, "/customers")
}
func (n *SpringBootSample) MeasureLatencyOnPath(expectedStatus int, path string) (time.Duration, error) {
	client, err := n.Minikube.NewRestClientForService(n.Service)
	if err != nil {
		return 0, err
	}
	defer client.Close()

	response, err := client.R().Get(path)
	if err != nil {
		return 0, err
	}
	if response.StatusCode() != expectedStatus {
		return 0, fmt.Errorf("unexpected status code %d, expected %d", response.StatusCode(), expectedStatus)
	}
	return response.Time(), nil
}

func (n *SpringBootSample) MeasureUnaffectedLatencyOnPath(expectedStatus int, path string) (time.Duration, error) {
	measurements := make([]time.Duration, 3)
	for i := 0; i < 3; i++ {
		latency, err := n.MeasureLatencyOnPath(expectedStatus, path)
		if err != nil {
			return 0, err
		}
		measurements[i] = latency
	}
	// median of measurements
	sum := 0 * time.Millisecond
	for _, measurement := range measurements {
		sum += measurement
	}
	return sum / 3, nil
}

func (n *SpringBootSample) AssertLatency(t *testing.T, min time.Duration, max time.Duration, unaffectedLatency time.Duration) {
	n.AssertLatencyOnPath(t, min, max, "/customers", unaffectedLatency)
}
func (n *SpringBootSample) AssertLatencyOnPath(t *testing.T, min time.Duration, max time.Duration, path string, unaffectedLatency time.Duration) {
	t.Helper()

	measurements := make([]time.Duration, 0, 5)
	e2e.Retry(t, 8, 500*time.Millisecond, func(r *e2e.R) {
		latency, err := n.MeasureLatencyOnPath(200, path)
		if err != nil {
			r.Failed = true
			_, _ = fmt.Fprintf(r.Log, "failed to measure package latency: %s", err)
		}
		if latency < min || latency > max {
			r.Failed = true
			measurements = append(measurements, latency)
			_, _ = fmt.Fprintf(r.Log, "package latency %v is not in expected range [%s, %s] with unaffectedLatency: %s", measurements, min, max, unaffectedLatency)
		}
	})
}

func (n *SpringBootSample) ExpectedStatus(expectedStatus int) (bool, int, error) {
	return n.ExpectedStatusOnPath(expectedStatus, "/customers")
}
func (n *SpringBootSample) ExpectedStatusOnPath(expectedStatus int, path string) (bool, int, error) {
	client, err := n.Minikube.NewRestClientForService(n.Service)
	if err != nil {
		return false, -1, err
	}
	defer client.Close()

	response, err := client.R().Get(path)
	if err != nil {
		return false, response.StatusCode(), err
	}
	if response.StatusCode() != expectedStatus {
		return false, response.StatusCode(), errors.New("unexpected status code")
	}
	return true, response.StatusCode(), nil
}
func (n *SpringBootSample) AssertStatus(t *testing.T, expectedHttpStatus int) {
	n.AssertStatusOnPath(t, expectedHttpStatus, "/customers")
}
func (n *SpringBootSample) AssertStatusOnPath(t *testing.T, expectedHttpStatus int, path string) {
	t.Helper()

	measurements := make([]int, 0, 5)
	e2e.Retry(t, 8, 500*time.Millisecond, func(r *e2e.R) {
		success, statusCode, err := n.ExpectedStatusOnPath(expectedHttpStatus, path)
		if err != nil {
			r.Failed = true
			_, _ = fmt.Fprintf(r.Log, "failed to get http status: %s", err)
		}
		if !success {
			r.Failed = true
			measurements = append(measurements, statusCode)
			_, _ = fmt.Fprintf(r.Log, "http status %v is not expected [%d]", measurements, expectedHttpStatus)
		}
	})
}

func (n *SpringBootSample) ContainerStatus() (*corev1.ContainerStatus, error) {
	return e2e.GetContainerStatus(n.Minikube, n.Pod, "spring-boot-sample")
}

func (n *SpringBootSample) Delete() error {
	return errors.Join(
		n.Minikube.DeletePod(n.Pod),
		n.Minikube.DeleteService(n.Service),
	)
}
