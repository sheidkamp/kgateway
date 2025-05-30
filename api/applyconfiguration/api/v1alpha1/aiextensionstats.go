// Code generated by applyconfiguration-gen. DO NOT EDIT.

package v1alpha1

import (
	apiv1alpha1 "github.com/kgateway-dev/kgateway/v2/api/v1alpha1"
)

// AiExtensionStatsApplyConfiguration represents a declarative configuration of the AiExtensionStats type for use
// with apply.
type AiExtensionStatsApplyConfiguration struct {
	CustomLabels []*apiv1alpha1.CustomLabel `json:"customLabels,omitempty"`
}

// AiExtensionStatsApplyConfiguration constructs a declarative configuration of the AiExtensionStats type for use with
// apply.
func AiExtensionStats() *AiExtensionStatsApplyConfiguration {
	return &AiExtensionStatsApplyConfiguration{}
}

// WithCustomLabels adds the given value to the CustomLabels field in the declarative configuration
// and returns the receiver, so that objects can be build by chaining "With" function invocations.
// If called multiple times, values provided by each call will be appended to the CustomLabels field.
func (b *AiExtensionStatsApplyConfiguration) WithCustomLabels(values ...**apiv1alpha1.CustomLabel) *AiExtensionStatsApplyConfiguration {
	for i := range values {
		if values[i] == nil {
			panic("nil value passed to WithCustomLabels")
		}
		b.CustomLabels = append(b.CustomLabels, *values[i])
	}
	return b
}
