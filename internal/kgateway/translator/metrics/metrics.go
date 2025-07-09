package metrics

import (
	"sync"
	"time"

	"github.com/kgateway-dev/kgateway/v2/pkg/metrics"
)

const (
	translatorSubsystem = "translator"
	translatorNameLabel = "translator"
	resourcesSubsystem  = "resources"
)

var (
	translationHistogramBuckets = []float64{0.0001, 0.00025, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1}
	translationsTotal           = metrics.NewCounter(
		metrics.CounterOpts{
			Subsystem: translatorSubsystem,
			Name:      "translations_total",
			Help:      "Total number of translations",
		},
		[]string{translatorNameLabel, "result"},
	)
	translationDuration = metrics.NewHistogram(
		metrics.HistogramOpts{
			Subsystem:                       translatorSubsystem,
			Name:                            "translation_duration_seconds",
			Help:                            "Translation duration",
			Buckets:                         translationHistogramBuckets,
			NativeHistogramBucketFactor:     1.1,
			NativeHistogramMaxBucketNumber:  100,
			NativeHistogramMinResetDuration: time.Hour,
		},
		[]string{translatorNameLabel},
	)
	translationsRunning = metrics.NewGauge(
		metrics.GaugeOpts{
			Subsystem: translatorSubsystem,
			Name:      "translations_running",
			Help:      "Current number of translations running",
		},
		[]string{translatorNameLabel},
	)

	resourcesSyncsStartedTotal = metrics.NewCounter(metrics.CounterOpts{
		Subsystem: resourcesSubsystem,
		Name:      "syncs_started_total",
		Help:      "Total number of syncs started",
	},
		[]string{"gateway", "namespace", "resource"})
)

type ResourceMetricLabels struct {
	Gateway   string
	Namespace string
	Resource  string
}

func (r ResourceMetricLabels) toMetricsLabels() []metrics.Label {
	return []metrics.Label{
		{Name: "gateway", Value: r.Gateway},
		{Name: "namespace", Value: r.Namespace},
		{Name: "resource", Value: r.Resource},
	}
}

// TranslatorMetricsRecorder defines the interface for recording translator metrics.
type TranslatorMetricsRecorder interface {
	TranslationStart() func(error)
}

// translatorMetrics records metrics for translator operations.
type translatorMetrics struct {
	translatorName      string
	translationsTotal   metrics.Counter
	translationDuration metrics.Histogram
	translationsRunning metrics.Gauge
}

// NewTranslatorMetricsRecorder creates a new recorder for translator metrics.
func NewTranslatorMetricsRecorder(translatorName string) TranslatorMetricsRecorder {
	if !metrics.Active() {
		return &nullTranslatorMetricsRecorder{}
	}

	m := &translatorMetrics{
		translatorName:      translatorName,
		translationsTotal:   translationsTotal,
		translationDuration: translationDuration,
		translationsRunning: translationsRunning,
	}

	return m
}

// TranslationStart is called at the start of a translation function to begin metrics
// collection and returns a function called at the end to complete metrics recording.
func (m *translatorMetrics) TranslationStart() func(error) {
	start := time.Now()

	m.translationsRunning.Add(1,
		metrics.Label{Name: translatorNameLabel, Value: m.translatorName})

	return func(err error) {
		duration := time.Since(start)

		m.translationDuration.Observe(duration.Seconds(),
			metrics.Label{Name: translatorNameLabel, Value: m.translatorName})

		result := "success"
		if err != nil {
			result = "error"
		}

		m.translationsTotal.Inc([]metrics.Label{
			{Name: translatorNameLabel, Value: m.translatorName},
			{Name: "result", Value: result},
		}...)

		m.translationsRunning.Sub(1,
			metrics.Label{Name: translatorNameLabel, Value: m.translatorName})
	}
}

var _ TranslatorMetricsRecorder = &translatorMetrics{}

type nullTranslatorMetricsRecorder struct{}

func (m *nullTranslatorMetricsRecorder) TranslationStart() func(error) {
	return func(err error) {}
}

// IncTranslationsTotal increments the total translations counter.
func IncResourcesSyncsStartedTotal(resourceName string, labels ResourceMetricLabels) {
	if StartResourceSync(ResourceSyncDetails{
		Namespace:    labels.Namespace,
		Gateway:      labels.Gateway,
		ResourceType: labels.Resource,
		ResourceName: resourceName,
	}) {
		resourcesSyncsStartedTotal.Inc(labels.toMetricsLabels()...)
	}
}

// ResourceSyncStartTime represents the start time of a resource sync.
type ResourceSyncStartTime struct {
	Time         time.Time
	ResourceType string
	ResourceName string
	Namespace    string
	Gateway      string
}

// resourceSyncStartTimes tracks the start times of resource syncs.
type resourceSyncStartTimes struct {
	sync.RWMutex
	times map[string]map[string]map[string][]ResourceSyncStartTime
}

var startTimes = &resourceSyncStartTimes{}

// ResourceSyncDetails holds the details of a resource sync operation.
type ResourceSyncDetails struct {
	Namespace    string
	Gateway      string
	ResourceType string
	ResourceName string
}

// StartResourceSync records the start time of a sync for a given resource key.
func StartResourceSync(details ResourceSyncDetails) bool {
	startTimes.Lock()
	defer startTimes.Unlock()

	if startTimes.times == nil {
		startTimes.times = make(map[string]map[string]map[string][]ResourceSyncStartTime)
	}

	if startTimes.times[details.Gateway] == nil {
		startTimes.times[details.Gateway] = make(map[string]map[string][]ResourceSyncStartTime)
	}

	if startTimes.times[details.Gateway][details.Namespace] == nil {
		startTimes.times[details.Gateway][details.Namespace] = make(map[string][]ResourceSyncStartTime)
	}

	if startTimes.times[details.Gateway][details.Namespace][details.ResourceType] == nil {
		startTimes.times[details.Gateway][details.Namespace][details.ResourceType] = []ResourceSyncStartTime{}
	}

	if startTimes.times[details.Gateway][details.Namespace]["XDSSnapshot"] == nil {
		startTimes.times[details.Gateway][details.Namespace]["XDSSnapshot"] = []ResourceSyncStartTime{}
	}

	st := ResourceSyncStartTime{
		Time:         time.Now(),
		ResourceType: details.ResourceType,
		ResourceName: details.ResourceName,
		Namespace:    details.Namespace,
		Gateway:      details.Gateway,
	}

	startTimes.times[details.Gateway][details.Namespace]["XDSSnapshot"] = append(startTimes.times[details.Gateway][details.Namespace]["XDSSnapshot"], st)
	startTimes.times[details.Gateway][details.Namespace][details.ResourceType] = append(startTimes.times[details.Gateway][details.Namespace][details.ResourceType], st)

	return true
}

// EndResourceSync removes the start time of a sync for a given resource key.
func EndResourceSync(
	details ResourceSyncDetails,
	xdsSnapshot bool,
	totalCounter metrics.Counter,
	durationHistogram metrics.Histogram,
) {
	startTimes.Lock()
	defer startTimes.Unlock()

	if startTimes.times == nil {
		return
	}

	if startTimes.times[details.Gateway] == nil {
		return
	}

	if startTimes.times[details.Gateway][details.Namespace] == nil {
		return
	}

	rt := details.ResourceType
	if xdsSnapshot {
		rt = "XDSSnapshot"
	} else {
		if startTimes.times[details.Gateway][details.Namespace][rt] == nil {
			return
		}
	}

	newSTs := []ResourceSyncStartTime{}
	res := []ResourceSyncStartTime{}

	if xdsSnapshot {
		for ns, stm := range startTimes.times[details.Gateway] {
			res = append(res, stm["XDSSnapshot"]...)

			delete(startTimes.times[details.Gateway][ns], "XDSSnapshot")
			if len(startTimes.times[details.Gateway][ns]) == 0 {
				delete(startTimes.times[details.Gateway], ns)
			}

			if len(startTimes.times[details.Gateway]) == 0 {
				delete(startTimes.times, details.Gateway)
			}
		}
	} else {
		for _, st := range startTimes.times[details.Gateway][details.Namespace][rt] {
			if st.ResourceType == details.ResourceType && st.ResourceName == details.ResourceName {
				res = append(res, ResourceSyncStartTime{
					Time:         st.Time,
					ResourceType: st.ResourceType,
					ResourceName: st.ResourceName,
					Namespace:    st.Namespace,
					Gateway:      st.Gateway,
				})

				continue
			}

			newSTs = append(newSTs, st)
		}

		startTimes.times[details.Gateway][details.Namespace][rt] = newSTs
		if len(startTimes.times[details.Gateway][details.Namespace][rt]) == 0 {
			delete(startTimes.times[details.Gateway][details.Namespace], rt)
		}

		if len(startTimes.times[details.Gateway][details.Namespace]) == 0 {
			delete(startTimes.times[details.Gateway], details.Namespace)
		}

		if len(startTimes.times[details.Gateway]) == 0 {
			delete(startTimes.times, details.Gateway)
		}
	}

	for _, st := range res {
		totalCounter.Inc([]metrics.Label{
			{Name: "gateway", Value: st.Gateway},
			{Name: "namespace", Value: st.Namespace},
			{Name: "resource", Value: st.ResourceType},
		}...)

		durationHistogram.Observe(time.Since(st.Time).Seconds(),
			[]metrics.Label{
				{Name: "gateway", Value: st.Gateway},
				{Name: "namespace", Value: st.Namespace},
				{Name: "resource", Value: st.ResourceType},
			}...)
	}
}

// ResetMetrics resets the metrics from this package.
// This is provided for testing purposes only.
func ResetMetrics() {
	translationsTotal.Reset()
	translationDuration.Reset()
	translationsRunning.Reset()
	resourcesSyncsStartedTotal.Reset()

	startTimes.Lock()
	defer startTimes.Unlock()
	startTimes.times = make(map[string]map[string]map[string][]ResourceSyncStartTime)
}
