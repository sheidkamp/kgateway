package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	routeSubsystem = "route"
)

var (
	routeListenerCounts = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Subsystem: routeSubsystem,
			Name:      "listener_counts",
			Help:      "Number of listeners for a route",
		},
		[]string{"name", "namespace", "type"},
	)
)

type RouteRecorder interface {
	SetListenerCount(route client.Object, routeType string, listeners int)
}

var _ RouteRecorder = &RouteMetrics{}

// TranslatorMetrics provides metrics for translator operations.
type RouteMetrics struct {
	routeListenerCounts *prometheus.GaugeVec
}

func NewRouteRecorder() RouteMetrics {
	return RouteMetrics{
		routeListenerCounts: routeListenerCounts,
	}
}

func (m *RouteMetrics) SetListenerCount(route client.Object, routeType string, listeners int) {
	m.routeListenerCounts.WithLabelValues(route.GetName(), route.GetNamespace(), routeType).Set(float64(listeners))
}
