package listenerpolicy

import (
	"context"

	"istio.io/istio/pkg/kube/krt"

	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/collections"
	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/ir"
)

// ListenerPolicyGatewayExtensionIR is the IR form of a GatewayExtension when applied to ListenerPolicy.
// This is currently minimal; expand as listener extensions are added.
type ListenerPolicyGatewayExtensionIR struct {
	Name             string
	PrecedenceWeight int32
	Err              error
}

var (
	_ krt.ResourceNamer                             = ListenerPolicyGatewayExtensionIR{}
	_ krt.Equaler[ListenerPolicyGatewayExtensionIR] = ListenerPolicyGatewayExtensionIR{}
)

// ResourceName returns the unique name for this extension.
func (e ListenerPolicyGatewayExtensionIR) ResourceName() string {
	return e.Name
}

// Equals compares two ListenerPolicyGatewayExtensionIR instances.
func (e ListenerPolicyGatewayExtensionIR) Equals(other ListenerPolicyGatewayExtensionIR) bool {
	if e.PrecedenceWeight != other.PrecedenceWeight {
		return false
	}

	if e.Err == nil && other.Err != nil {
		return false
	}
	if e.Err != nil && other.Err == nil {
		return false
	}
	if e.Err != nil && other.Err != nil && e.Err.Error() != other.Err.Error() {
		return false
	}
	return true
}

// Validate performs validation on the extension IR.
func (e ListenerPolicyGatewayExtensionIR) Validate() error {
	// Nothing to validate yet; add checks as extension fields are introduced.
	return nil
}

// TranslateGatewayExtensionBuilder builds a translator that converts GatewayExtension resources
// into ListenerPolicyGatewayExtensionIR. Currently minimal since listener policies do not
// consume gateway extensions.
func TranslateGatewayExtensionBuilder(
	_ context.Context,
	_ *collections.CommonCollections,
) func(krtctx krt.HandlerContext, gExt ir.GatewayExtension) *ListenerPolicyGatewayExtensionIR {
	return func(_ krt.HandlerContext, gExt ir.GatewayExtension) *ListenerPolicyGatewayExtensionIR {
		return &ListenerPolicyGatewayExtensionIR{
			Name:             gExt.ResourceName(),
			PrecedenceWeight: gExt.PrecedenceWeight,
		}
	}
}
