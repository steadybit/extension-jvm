// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"errors"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
	"github.com/steadybit/extension-kit/extutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	acorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	ametav1 "k8s.io/client-go/applyconfigurations/meta/v1"
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
					Image: extutil.Ptr("steadybit/spring-boot-sample:1.0.20"),
					Ports: []acorev1.ContainerPortApplyConfiguration{
						{
							ContainerPort: extutil.Ptr(int32(80)),
						},
					},
          Env: []acorev1.EnvVarApplyConfiguration{
            {
              Name: extutil.Ptr("STEADYBIT_LOG_JAVAAGENT_STDOUT"),
              Value: extutil.Ptr("true"),
            },
            {
              Name: extutil.Ptr("STEADYBIT_LOG_LEVEL"),
              Value: extutil.Ptr("TRACE"),
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
					Port:     extutil.Ptr(int32(80)),
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

	_, err = client.R().Get("/")
	if err != nil {
		return err
	}

	return nil
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