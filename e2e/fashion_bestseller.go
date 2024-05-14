// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: 2023 Steadybit GmbH

package e2e

import (
	"errors"
	"github.com/steadybit/action-kit/go/action_kit_api/v2"
	"github.com/steadybit/action-kit/go/action_kit_test/e2e"
	"github.com/steadybit/extension-kit/extutil"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	acorev1 "k8s.io/client-go/applyconfigurations/core/v1"
	ametav1 "k8s.io/client-go/applyconfigurations/meta/v1"
)

type FashionBestseller struct {
	Minikube  *e2e.Minikube
	Pod       metav1.Object
	Service   metav1.Object
	cancelCtx context.CancelFunc
}

func (n *FashionBestseller) Deploy(podName string, opts ...func(c *acorev1.PodApplyConfiguration)) error {
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
					Name:  extutil.Ptr("fashion-bestseller"),
					Image: extutil.Ptr("docker.io/steadybit/bestseller-fashion:main"),
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
			Name:   extutil.Ptr("fashion-bestseller"),
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

	ctx, cancel := context.WithCancel(context.Background())
	n.cancelCtx = cancel
	go n.Minikube.TailLog(ctx, n.Pod)

	return nil
}

func (n *FashionBestseller) Target() (*action_kit_api.Target, error) {
	return e2e.NewContainerTarget(n.Minikube, n.Pod, "fashion-bestseller")
}

func (n *FashionBestseller) IsReachable() error {
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

func (n *FashionBestseller) ContainerStatus() (*corev1.ContainerStatus, error) {
	return e2e.GetContainerStatus(n.Minikube, n.Pod, "fashion-bestseller")
}

func (n *FashionBestseller) Delete() error {
	n.cancelCtx()
	return errors.Join(
		n.Minikube.DeletePod(n.Pod),
		n.Minikube.DeleteService(n.Service),
	)
}
