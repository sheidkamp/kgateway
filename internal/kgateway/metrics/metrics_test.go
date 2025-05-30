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
	controllerResourcesManaged.Reset()
	translationsTotal.Reset()
	translationDuration.Reset()
	translatorResourcesManaged.Reset()
}
