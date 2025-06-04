# Metrics Package

The purpose of this packgage is to encapsulate the implementation of the control plane metrics.

- All metrics are defined and initialized at package level and not exported.
- The package currently uses "github.com/prometheus/client_golang" as the metrics backend.
- All metrics are initialized and registered with the controller-runtime regsistry with the init process in [metrics.go](metrics.go).
- Constants are used to ensure metric naming conventions are followed for namespaces and subsystems. And, the following naming conventions are used for metrics and labels: https://prometheus.io/docs/practices/naming/.
- Metrics recorder interfaces are exported, with functions to create them.
- These interfaces are the way metrics are computed or interacted with in any code outside of this package.
- This allows flexibility if support for additional or different metrics backed libraries is ever needed in the future. Backend metrics implementations can be changed, as long as the exported interfaces remain intact.
- The following interfaces are currently exported for metrics computation in other packages:
  - CollectionRecorder: recording metrics related to KRT collections and transform operations.
  - ControllerRecorder: recording metrics related to kubernetes controllers and reconcile operations.
  - StatusSyncerRecorder: recording metrics related to resource status syncer operations.
  - TranslatorRecorder: recording metrics related to IR and XDS translation operations.
