package gwapiutils

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver/v3"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsv1client "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GwApiChannel represents the Gateway API release channel
type GwApiChannel string

const (
	GwApiChannelStandard     GwApiChannel = "standard"
	GwApiChannelExperimental GwApiChannel = "experimental"
)

// GwApiVersion wraps semver.Version for Gateway API versions
type GwApiVersion struct {
	Version semver.Version
}

// GwApiInfo contains the detected Gateway API channel and version
type GwApiInfo struct {
	Channel GwApiChannel
	Version *GwApiVersion
}

// DetectGatewayAPIVersion reads the Gateway CRD to determine the installed
// Gateway API channel and version from annotations.
// It looks for the following annotations on the gateways.gateway.networking.k8s.io CRD:
//   - gateway.networking.k8s.io/channel: the API channel (standard or experimental)
//   - gateway.networking.k8s.io/bundle-version: the API version (e.g., v1.2.0)
func DetectGatewayAPIVersion(ctx context.Context, c client.Client) (*GwApiInfo, error) {
	var gatewayCRD apiextensionsv1.CustomResourceDefinition
	err := c.Get(ctx, client.ObjectKey{Name: "gateways.gateway.networking.k8s.io"}, &gatewayCRD)
	if err != nil {
		return nil, fmt.Errorf("failed to get Gateway CRD: %w", err)
	}

	channel, hasChannel := gatewayCRD.Annotations["gateway.networking.k8s.io/channel"]
	if !hasChannel {
		return nil, fmt.Errorf("Gateway CRD missing 'gateway.networking.k8s.io/channel' annotation")
	}

	versionStr, hasVersion := gatewayCRD.Annotations["gateway.networking.k8s.io/bundle-version"]
	if !hasVersion {
		return nil, fmt.Errorf("Gateway CRD missing 'gateway.networking.k8s.io/bundle-version' annotation")
	}

	version, err := semver.NewVersion(versionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Gateway API version %q: %w", versionStr, err)
	}

	return &GwApiInfo{
		Channel: GwApiChannel(channel),
		Version: &GwApiVersion{Version: *version},
	}, nil
}

// DetectGatewayAPIVersionWithClient reads the Gateway CRD using an apiextensions client.
// This is useful when you need to read CRDs before the controller-runtime cache is started.
func DetectGatewayAPIVersionWithClient(ctx context.Context, c apiextensionsv1client.CustomResourceDefinitionInterface) (*GwApiInfo, error) {
	gatewayCRD, err := c.Get(ctx, "gateways.gateway.networking.k8s.io", metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Gateway CRD: %w", err)
	}

	channel, hasChannel := gatewayCRD.Annotations["gateway.networking.k8s.io/channel"]
	if !hasChannel {
		return nil, fmt.Errorf("Gateway CRD missing 'gateway.networking.k8s.io/channel' annotation")
	}

	versionStr, hasVersion := gatewayCRD.Annotations["gateway.networking.k8s.io/bundle-version"]
	if !hasVersion {
		return nil, fmt.Errorf("Gateway CRD missing 'gateway.networking.k8s.io/bundle-version' annotation")
	}

	version, err := semver.NewVersion(versionStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Gateway API version %q: %w", versionStr, err)
	}

	return &GwApiInfo{
		Channel: GwApiChannel(channel),
		Version: &GwApiVersion{Version: *version},
	}, nil
}

// IsStandard returns true if the channel is standard
func (c GwApiChannel) IsStandard() bool {
	return c == GwApiChannelStandard
}

// IsExperimental returns true if the channel is experimental
func (c GwApiChannel) IsExperimental() bool {
	return c == GwApiChannelExperimental
}

// String returns the string representation of the version
func (v *GwApiVersion) String() string {
	return v.Version.String()
}
