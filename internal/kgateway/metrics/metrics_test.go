package metrics

import (
	"testing"
)

func TestMain(m *testing.M) {
	m.Run()
}

func setupTest() {
	reconciliationsTotal.Reset()
	reconcileDuration.Reset()
	translationsTotal.Reset()
	translationDuration.Reset()
	translatorResourceCount.Reset()
	snapshotSyncsTotal.Reset()
	snapshotSyncDuration.Reset()
	snapshotResourceCount.Reset()
}
