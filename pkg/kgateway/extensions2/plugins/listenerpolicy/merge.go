package listenerpolicy

import (
	"encoding/json"
	"fmt"

	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/ir"
	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/policy"
)

type listenerMergeOpts struct {
	ListenerPolicy ListenerPolicyMergeOpts `json:"listenerPolicy,omitempty"`
}

// ListenerPolicyMergeOpts holds merge-time overrides for listener policies.
// Currently empty; add fields as merge customization is needed.
type ListenerPolicyMergeOpts struct{}

func MergePolicies(
	p1, p2 *ListenerPolicyIR,
	p2Ref *ir.AttachedPolicyRef,
	p2MergeOrigins ir.MergeOrigins,
	mergeOpts policy.MergeOptions,
	mergeOrigins ir.MergeOrigins,
	mergeSettingsJSON string,
) {
	if p1 == nil || p2 == nil {
		return
	}
	if p1 != nil && p2 != nil {
		if p1.NoOrigin || p2.NoOrigin {
			p1.NoOrigin = true
			p2.NoOrigin = true
		}
	}

	var polMergeOpts listenerMergeOpts
	if mergeSettingsJSON != "" {
		if err := json.Unmarshal([]byte(mergeSettingsJSON), &polMergeOpts); err != nil {
			logger.Error("error parsing merge settings; skipping merge", "value", mergeSettingsJSON, "error", err)
		}
	}
	lpOpts := polMergeOpts.ListenerPolicy

	mergeFuncs := []func(*ListenerPolicyIR, *ListenerPolicyIR, *ir.AttachedPolicyRef, ir.MergeOrigins, policy.MergeOptions, ir.MergeOrigins, ListenerPolicyMergeOpts){
		mergeDefault,
		mergePerPort,
	}

	for _, mergeFunc := range mergeFuncs {
		mergeFunc(p1, p2, p2Ref, p2MergeOrigins, mergeOpts, mergeOrigins, lpOpts)
	}
}

func mergeDefault(
	p1, p2 *ListenerPolicyIR,
	p2Ref *ir.AttachedPolicyRef,
	p2MergeOrigins ir.MergeOrigins,
	opts policy.MergeOptions,
	mergeOrigins ir.MergeOrigins,
	lpOpts ListenerPolicyMergeOpts,
) {
	origin := "default."
	if (p1 != nil && p1.NoOrigin) || (p2 != nil && p2.NoOrigin) {
		origin = ""
	}
	mergeListenerPolicy(origin, &p1.defaultPolicy, &p2.defaultPolicy, p2Ref, p2MergeOrigins, opts, mergeOrigins, lpOpts)
}

func mergePerPort(
	p1, p2 *ListenerPolicyIR,
	p2Ref *ir.AttachedPolicyRef,
	p2MergeOrigins ir.MergeOrigins,
	opts policy.MergeOptions,
	mergeOrigins ir.MergeOrigins,
	lpOpts ListenerPolicyMergeOpts,
) {
	for port, p2PortPolicy := range p2.perPortPolicy {
		if p1PortPolicy, ok := p1.perPortPolicy[port]; ok {
			f := fmt.Sprintf("perPortPolicy[%d].", port)
			if (p1 != nil && p1.NoOrigin) || (p2 != nil && p2.NoOrigin) {
				f = ""
			}
			mergeListenerPolicy(f, &p1PortPolicy, &p2PortPolicy, p2Ref, p2MergeOrigins, opts, mergeOrigins, lpOpts)
			p1.perPortPolicy[port] = p1PortPolicy
		} else {
			f := fmt.Sprintf("perPortPolicy[%d]", port)
			if (p1 != nil && p1.NoOrigin) || (p2 != nil && p2.NoOrigin) {
				f = ""
			}
			if p1.perPortPolicy == nil {
				p1.perPortPolicy = map[uint32]ListenerPolicy{}
			}
			p1.perPortPolicy[port] = p2PortPolicy
			mergeOrigins.SetOne(f, p2Ref, p2MergeOrigins)
		}
	}
}

func mergeListenerPolicy(
	origin string,
	p1, p2 *ListenerPolicy,
	p2Ref *ir.AttachedPolicyRef,
	p2MergeOrigins ir.MergeOrigins,
	mergeOpts policy.MergeOptions,
	mergeOrigins ir.MergeOrigins,
	lpOpts ListenerPolicyMergeOpts,
) {
	mergeFuncs := []func(string, *ListenerPolicy, *ListenerPolicy, *ir.AttachedPolicyRef, ir.MergeOrigins, policy.MergeOptions, ir.MergeOrigins, ListenerPolicyMergeOpts){
		mergeProxyProtocol,
		mergePerConnectionBufferLimitBytes,
		mergeHttpSettings,
	}

	for _, mergeFunc := range mergeFuncs {
		mergeFunc(origin, p1, p2, p2Ref, p2MergeOrigins, mergeOpts, mergeOrigins, lpOpts)
	}
}

func mergeProxyProtocol(
	origin string,
	p1, p2 *ListenerPolicy,
	p2Ref *ir.AttachedPolicyRef,
	p2MergeOrigins ir.MergeOrigins,
	opts policy.MergeOptions,
	mergeOrigins ir.MergeOrigins,
	_ ListenerPolicyMergeOpts,
) {
	if !policy.IsMergeable(p1.proxyProtocol, p2.proxyProtocol, opts) {
		return
	}

	p1.proxyProtocol = p2.proxyProtocol
	mergeOrigins.SetOne(origin+"proxyProtocol", p2Ref, p2MergeOrigins)
}

func mergePerConnectionBufferLimitBytes(
	origin string,
	p1, p2 *ListenerPolicy,
	p2Ref *ir.AttachedPolicyRef,
	p2MergeOrigins ir.MergeOrigins,
	opts policy.MergeOptions,
	mergeOrigins ir.MergeOrigins,
	_ ListenerPolicyMergeOpts,
) {
	if !policy.IsMergeable(p1.perConnectionBufferLimitBytes, p2.perConnectionBufferLimitBytes, opts) {
		return
	}

	p1.perConnectionBufferLimitBytes = p2.perConnectionBufferLimitBytes
	mergeOrigins.SetOne(origin+"perConnectionBufferLimitBytes", p2Ref, p2MergeOrigins)
}
func mergeHttpSettings(
	origin string,
	p1, p2 *ListenerPolicy,
	p2Ref *ir.AttachedPolicyRef,
	p2MergeOrigins ir.MergeOrigins,
	opts policy.MergeOptions,
	mergeOrigins ir.MergeOrigins,
	_ ListenerPolicyMergeOpts,
) {
	if p2.http == nil {
		return
	}
	if p1.http == nil {
		p1.http = &HttpListenerPolicyIr{}
	}
	if origin != "" {
		origin += "httpSettings."
	}
	MergeHttpPolicies(origin, p1.http, p2.http, p2Ref, p2MergeOrigins, opts, mergeOrigins, "" /*no merge settings*/)
}
