package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// +kubebuilder:rbac:groups=gateway.kgateway.dev,resources=mcproutes,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.kgateway.dev,resources=mcproutes/status,verbs=get;update;patch

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:metadata:labels={app=kgateway,app.kubernetes.io/name=kgateway}
// +kubebuilder:resource:categories=kgateway
// +kubebuilder:subresource:status
type MCPRoute struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MCPRouteSpec   `json:"spec,omitempty"`
	Status MCPRouteStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type MCPRouteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MCPRoute `json:"items"`
}

type MCPRouteSpec struct {
	ParentRefs []gwv1.ParentReference `json:"parentRefs,omitempty"`

	BackendRef gwv1.BackendRef `json:"backendRef,omitempty"`
}

type MCPRouteStatus struct {
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

/*
type MCPRule struct {
	Matches []MCPMatch `json:"matches,omitempty"`
	Route   MCPRoute   `json:"route,omitempty"`
}

type MCPMatch struct {
	Tool        string            `json:"tool,omitempty"`
	BackendRefs []gwv1.BackendRef `json:"backendRefs,omitempty"`
}

*/

// +kubebuilder:rbac:groups=gateway.kgateway.dev,resources=mcpauthpolicies,verbs=get;list;watch
// +kubebuilder:rbac:groups=gateway.kgateway.dev,resources=mcpauthpolicies/status,verbs=get;update;patch

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:metadata:labels={app=kgateway,app.kubernetes.io/name=kgateway}
// +kubebuilder:resource:categories=kgateway
// +kubebuilder:subresource:status
type MCPAuthPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MCPAuthPolicySpec   `json:"spec,omitempty"`
	Status MCPAuthPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type MCPAuthPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MCPAuthPolicy `json:"items"`
}

type MCPAuthPolicySpec struct {
	TargetRefs []LocalPolicyTargetReference `json:"targetRefs,omitempty"`

	Rules []MCPAuthPolicyRule `json:"rules,omitempty"`
}

type MCPAuthPolicyRule struct {
	Matches  []MCPAuthPolicyMatch `json:"matches,omitempty"`
	Resource MCPResource          `json:"resource,omitempty"`
}

type MCPAuthPolicyMatchType string

const (
	MCPAuthPolicyMatchTypeJWT MCPAuthPolicyMatchType = "jwt"
)

// match an incoming JWT token
type MCPAuthPolicyMatch struct {
	Type MCPAuthPolicyMatchType `json:"type,omitempty"`

	JWT            *MCPAuthPolicyMatchJWT            `json:"jwt,omitempty"`
	ServiceAccount *MCPAuthPolicyMatchServiceAccount `json:"serviceAccount,omitempty"`
}

type MCPAuthPolicyMatchJWT struct {
	Claim string `json:"claim,omitempty"`
	Value string `json:"value,omitempty"`
}

type MCPAuthPolicyMatchServiceAccount struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type MCPResource struct {
	Kind string `json:"kind,omitempty"`
	Name string `json:"name,omitempty"`
}

type MCPAuthPolicyStatus struct {
	//
	// +optional
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=8
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}
