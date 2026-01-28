package tests

import (
	"testing"

	"github.com/kgateway-dev/kgateway/v2/test/celvalidation"
)

func TestCRDs(t *testing.T) {
	v := NewKgatewayValidator(t)
	celvalidation.TestCRDValidation(t, v, "testdata")
}
