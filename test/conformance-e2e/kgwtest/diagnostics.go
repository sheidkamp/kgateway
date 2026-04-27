package kgwtest

import "testing"

// failureHook runs after every subtest and logs a pointer for diagnostics
// when the test failed. The full dump implementation (controller logs, Envoy
// config-dump, resource YAMLs from the per-test namespace) lives behind this
// hook so it can be extended without touching call sites.
func failureHook(t *testing.T, test *Test, ns string) {
	if !t.Failed() {
		return
	}
	t.Logf("kgwtest: test %q failed — collect diagnostics from namespace %q and the kgateway controller pod", test.ShortName, ns)
}
