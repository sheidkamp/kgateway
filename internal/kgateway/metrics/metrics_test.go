package metrics_test

import (
	. "github.com/kgateway-dev/kgateway/v2/internal/kgateway/metrics"
)

func setupTest() {
	GetTransformDuration().Reset()
	GetTransformsTotal().Reset()
	GetCollectionResources().Reset()
	GetReconcileDuration().Reset()
	GetReconciliationsTotal().Reset()
	GetStatusSyncDuration().Reset()
	GetStatusSyncsTotal().Reset()
	GetStatusSyncResources().Reset()
	GetTranslationDuration().Reset()
	GetTranslationsTotal().Reset()
	GetDomainsPerListener().Reset()
}
