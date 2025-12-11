package listenerpolicy

import (
	"context"

	"istio.io/istio/pkg/kube/krt"

	"github.com/kgateway-dev/kgateway/v2/api/v1alpha1/kgateway"
	kgwwellknown "github.com/kgateway-dev/kgateway/v2/pkg/kgateway/wellknown"
	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/collections"
	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/ir"
)

// ListenerPolicyConstructor converts a ListenerPolicy CR into its IR form.
type ListenerPolicyConstructor struct {
	commoncol *collections.CommonCollections
}

// NewListenerPolicyConstructor creates a new ListenerPolicyConstructor.
func NewListenerPolicyConstructor(
	_ context.Context,
	commoncol *collections.CommonCollections,
) *ListenerPolicyConstructor {
	return &ListenerPolicyConstructor{
		commoncol: commoncol,
	}
}

// ConstructIR builds the ListenerPolicy IR for the given policy CR.
func (c *ListenerPolicyConstructor) ConstructIR(
	krtctx krt.HandlerContext,
	policyCR *kgateway.ListenerPolicy,
) (*ListenerPolicyIR, []error) {
	if policyCR == nil {
		return nil, nil
	}

	objSrc := ir.ObjectSource{
		Group:     kgwwellknown.ListenerPolicyGVK.Group,
		Kind:      kgwwellknown.ListenerPolicyGVK.Kind,
		Namespace: policyCR.Namespace,
		Name:      policyCR.Name,
	}

	return NewListenerPolicyIR(
		krtctx,
		c.commoncol,
		policyCR.CreationTimestamp.Time,
		&policyCR.Spec,
		objSrc,
	)
}

// HasSynced reports whether constructor-owned collections are synced.
func (c *ListenerPolicyConstructor) HasSynced() bool {
	// No additional collections beyond what the plugin already tracks.
	return true
}
