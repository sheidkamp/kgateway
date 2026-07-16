//go:build e2e

package common

import (
	"context"
	"fmt"
	"net/url"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// LookupLocalstackEndpoint resolves the NodePort URL for the localstack service.
// Returns ("", false, nil) if localstack is not installed on this cluster.
// Returns ("", false, err) on unexpected errors (failed list, parse failure, etc.).
func LookupLocalstackEndpoint(ctx context.Context, c client.Client) (string, bool, error) {
	svc := &corev1.Service{}
	err := c.Get(ctx, client.ObjectKey{Name: "localstack", Namespace: "localstack"}, svc)
	if apierrors.IsNotFound(err) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get localstack service: %w", err)
	}
	if len(svc.Spec.Ports) == 0 || svc.Spec.Ports[0].NodePort == 0 {
		return "", false, fmt.Errorf("localstack service has no NodePort")
	}
	port := svc.Spec.Ports[0].NodePort

	var nodeList corev1.NodeList
	if err := c.List(ctx, &nodeList); err != nil {
		return "", false, fmt.Errorf("list nodes: %w", err)
	}
	var nodeIP string
	for _, node := range nodeList.Items {
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				nodeIP = addr.Address
				break
			}
		}
		if nodeIP != "" {
			break
		}
	}
	if nodeIP == "" {
		return "", false, fmt.Errorf("no node with InternalIP found")
	}

	u, err := url.Parse(fmt.Sprintf("http://%s:%d", nodeIP, port))
	if err != nil {
		return "", false, fmt.Errorf("parse localstack URL: %w", err)
	}
	return u.String(), true, nil
}
