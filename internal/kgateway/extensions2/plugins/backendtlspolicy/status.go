package backendtlspolicy

import (
	"context"
	"fmt"

	"istio.io/istio/pkg/kube/kclient"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kgateway-dev/kgateway/v2/pkg/logging"
	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk"
)

var logger = logging.New("plugin/backendtlspolicy")

func getPolicyStatusFn(
	cl kclient.Client[*gwv1.BackendTLSPolicy],
) pluginsdk.GetPolicyStatusFn {
	return func(ctx context.Context, nn types.NamespacedName) (gwv1.PolicyStatus, error) {
		res := cl.Get(nn.Name, nn.Namespace)
		if res == nil {
			return gwv1.PolicyStatus{}, pluginsdk.ErrNotFound
		}
		return res.Status, nil
	}
}

func patchPolicyStatusFn(
	cl kclient.Client[*gwv1.BackendTLSPolicy],
) pluginsdk.PatchPolicyStatusFn {
	return func(ctx context.Context, nn types.NamespacedName, policyStatus gwv1.PolicyStatus) error {
		cur := cl.Get(nn.Name, nn.Namespace)
		if cur == nil {
			return pluginsdk.ErrNotFound
		}
		if _, err := cl.UpdateStatus(&gwv1.BackendTLSPolicy{
			ObjectMeta: pluginsdk.CloneObjectMetaForStatus(cur.ObjectMeta),
			Status:     policyStatus,
		}); err != nil {
			if errors.IsConflict(err) {
				logger.Debug("error updating stale status", "ref", nn, "error", err)
				return nil // let the conflicting Status update trigger a KRT event to requeue the updated object
			}
			return fmt.Errorf("error updating status for TrafficPolicy %s: %w", nn, err)
		}
		return nil
	}
}
