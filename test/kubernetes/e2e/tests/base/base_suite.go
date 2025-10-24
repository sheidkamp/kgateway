//go:build e2e

package base

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiserverschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"

	deployerinternal "github.com/kgateway-dev/kgateway/v2/internal/kgateway/deployer"
	"github.com/kgateway-dev/kgateway/v2/internal/kgateway/wellknown"
	"github.com/kgateway-dev/kgateway/v2/pkg/deployer"
	"github.com/kgateway-dev/kgateway/v2/test/kubernetes/e2e"
	"github.com/kgateway-dev/kgateway/v2/test/kubernetes/e2e/defaults"
	"github.com/kgateway-dev/kgateway/v2/test/testutils"
)

// GatewayApiChannel represents the Gateway API release channel
type GatewayApiChannel string

// Gateway API channel constants
const (
	GwApiChannelStandard     GatewayApiChannel = "standard"
	GwApiChannelExperimental GatewayApiChannel = "experimental"
)

// Named Gateway API version constants for easy reference
var (
	// GwApiV1_4_0 represents Gateway API version v1.4.0
	GwApiV1_4_0 = semver.MustParse("1.4.0")
)

// TestCase defines the manifests and resources used by a test or test suite.
type TestCase struct {
	// Manifests contains a list of manifest filenames.
	Manifests []string

	// ManifestsWithTransform maps a manifest filename to a function that transforms its contents before applying it
	ManifestsWithTransform map[string]func(string) string

	// manifestResources contains the resources automatically loaded from the manifest files for
	// this test case.
	manifestResources []client.Object

	// dynamicResources contains the expected dynamically provisioned resources for any Gateways
	// contained in this test case's manifests.
	dynamicResources []client.Object

	// GatewayApiVersion specifies the minimum Gateway API version required per channel.
	// Map key is the channel (GatewayApiChannelStandard or GatewayApiChannelExperimental), value is the minimum version.
	// If the map is empty/nil, the test runs on any channel/version.
	// Matching logic based on installed channel:
	//   - experimental: If experimental key exists, check version; otherwise run
	//   - standard: If standard key exists, check version; if only experimental exists, skip; otherwise runs .
	GatewayApiVersion map[GatewayApiChannel]*semver.Version
}

type BaseTestingSuite struct {
	suite.Suite
	Ctx              context.Context
	TestInstallation *e2e.TestInstallation
	Setup            TestCase
	TestCases        map[string]*TestCase

	// (Optional) Path of directory (relative to git root) containing the CRDs that will be used to read
	// the objects from the manifests. If empty then defaults to "install/helm/kgateway-crds/templates"
	CrdPath string

	// (Optional) Helper to determine if a Gateway is self-managed. If not provided, a default implementation
	// is used.
	GatewayHelper GatewayHelper

	// used internally to parse the manifest files
	gvkToStructuralSchema map[schema.GroupVersionKind]*apiserverschema.Structural

	// SetupByVersion allows defining different setup configurations for different GW API versions and channels.
	// Key is the TestCase to use, value is the channel/version requirements (same format as TestCase.GatewayApiVersion).
	// The system will select the setup with the highest matching version requirement for the current channel.
	// If no setups match, falls back to the Setup field.
	// Example:
	//   SetupByVersion: map[*TestCase]map[GatewayApiChannel]*semver.Version{
	//     &setupExperimental: {GatewayApiChannelExperimental: v1.4.0},
	//     &setupStandard: {GatewayApiChannelStandard: v1.4.0, GatewayApiChannelExperimental: v1.3.0},
	//   }
	SetupByVersion map[*TestCase]map[GatewayApiChannel]*semver.Version

	// selectedSetup tracks which setup was actually used, so we can clean it up in TearDownSuite
	selectedSetup *TestCase
}

// NewBaseTestingSuite returns a BaseTestingSuite that performs all the pre-requisites of upgrading helm installations,
// applying manifests and verifying resources exist before a suite and tests and the corresponding post-run cleanup.
// The pre-requisites for the suite are defined in the setup parameter and for each test in the individual testCase.
func NewBaseTestingSuite(ctx context.Context, testInst *e2e.TestInstallation, setupTestCase TestCase, testCases map[string]*TestCase) *BaseTestingSuite {
	suite := &BaseTestingSuite{
		Ctx:              ctx,
		TestInstallation: testInst,
		Setup:            setupTestCase,
		TestCases:        testCases,
	}

	return suite
}

// requirementsMatch checks if the given requirements match the current environment.
// This implements the matching logic for both setup selection and test skipping.
// Returns true if the requirements are satisfied by the current channel/version.
func (s *BaseTestingSuite) requirementsMatch(requirements map[GatewayApiChannel]*semver.Version, currentChannel GatewayApiChannel, currentVersion *semver.Version) bool {
	if len(requirements) == 0 {
		return true // No requirements = always matches
	}

	if currentChannel == GwApiChannelExperimental {
		// 1) If experimental version defined - compare versions
		if requiredVersion, hasExperimental := requirements[GwApiChannelExperimental]; hasExperimental {
			return currentVersion.GreaterThan(requiredVersion) || currentVersion.Equal(requiredVersion)
		}
		// 2) If no experimental requirement, check if it's standard-only (has standard but not experimental)
		if _, hasStandard := requirements[GwApiChannelStandard]; hasStandard {
			return false // This is standard-only, don't match experimental
		}
		// No requirements = matches any experimental
		return true
	}

	if currentChannel == GwApiChannelStandard {
		// 1) If standard version defined - compare versions
		if requiredVersion, hasStandard := requirements[GwApiChannelStandard]; hasStandard {
			return currentVersion.GreaterThan(requiredVersion) || currentVersion.Equal(requiredVersion)
		}
		// 2) If experimental defined but not standard - don't match (experimental-only)
		if _, hasExperimental := requirements[GwApiChannelExperimental]; hasExperimental {
			return false
		}
		// 3) No requirements = matches any standard
		return true
	}

	return false // Unknown channel
}

// selectSetup chooses the appropriate setup TestCase based on the current Gateway API version and channel.
// If SetupByVersion is defined, it selects the setup with the highest matching version requirement.
// Otherwise, it returns the default Setup.
func (s *BaseTestingSuite) selectSetup() *TestCase {
	// If versioned setups are not defined, use the default Setup
	if len(s.SetupByVersion) == 0 {
		return &s.Setup
	}

	currentVersion := s.getCurrentGatewayApiVersion()
	currentChannel := s.getCurrentGatewayApiChannel()

	if currentVersion == nil {
		// Can't determine version, fall back to default
		return &s.Setup
	}

	var bestMatch *TestCase
	var bestVersion *semver.Version

	// Find all matching setups and pick the most specific (highest version for current channel)
	for setup, requirements := range s.SetupByVersion {
		if !s.requirementsMatch(requirements, currentChannel, currentVersion) {
			continue // Requirements not met
		}

		// Track the version requirement for the current channel to find "best"
		var matchVersion *semver.Version
		if currentChannel == GwApiChannelExperimental {
			matchVersion = requirements[GwApiChannelExperimental]
		} else if currentChannel == GwApiChannelStandard {
			matchVersion = requirements[GwApiChannelStandard]
		}

		// Pick the setup with the highest version requirement (most specific)
		if matchVersion != nil {
			if bestVersion == nil || matchVersion.GreaterThan(bestVersion) {
				bestVersion = matchVersion
				bestMatch = setup
			}
		} else if bestMatch == nil {
			// This setup has no specific version requirement but matches channel
			bestMatch = setup
		}
	}

	if bestMatch != nil {
		return bestMatch
	}

	// Fallback to default Setup if no match
	return &s.Setup
}

func (s *BaseTestingSuite) SetupSuite() {
	// set up the helpers once and store them on the suite
	s.setupHelpers()

	// Select the appropriate setup based on Gateway API version
	s.selectedSetup = s.selectSetup()
	s.ApplyManifests(s.selectedSetup)
}

func (s *BaseTestingSuite) TearDownSuite() {
	if testutils.ShouldPersistInstall() {
		return
	}

	// Use the selected setup if available, otherwise fall back to default Setup
	setupToDelete := s.selectedSetup
	if setupToDelete == nil {
		setupToDelete = &s.Setup
	}
	s.DeleteManifests(setupToDelete)
}

func (s *BaseTestingSuite) BeforeTest(suiteName, testName string) {
	// apply test-specific manifests
	testCase, ok := s.TestCases[testName]
	if !ok {
		return
	}

	// Check version requirements before applying manifests
	if shouldSkip := s.shouldSkipTest(testCase); shouldSkip {
		s.T().Skipf("Test requires Gateway API %s, but current is %s/%s",
			testCase.GatewayApiVersion, s.getCurrentGatewayApiChannel(), s.getCurrentGatewayApiVersion())
		return
	}

	s.ApplyManifests(testCase)
}

func (s *BaseTestingSuite) AfterTest(suiteName, testName string) {
	// Delete test-specific manifests
	testCase, ok := s.TestCases[testName]
	if !ok {
		return
	}

	// Check if the test was skipped due to version requirements
	// If so, don't try to delete resources that were never applied
	if shouldSkip := s.shouldSkipTest(testCase); shouldSkip {
		return
	}

	if s.T().Failed() {
		s.TestInstallation.PreFailHandler(s.Ctx)
	}

	if testutils.ShouldPersistInstall() {
		return
	}
	s.DeleteManifests(testCase)
}

func (s *BaseTestingSuite) GetKubectlOutput(command ...string) string {
	out, _, err := s.TestInstallation.Actions.Kubectl().Execute(s.Ctx, command...)
	s.TestInstallation.Assertions.Require.NoError(err)

	return out
}

// ApplyManifests applies the manifests and waits until the resources are created and ready.
func (s *BaseTestingSuite) ApplyManifests(testCase *TestCase) {
	// apply the manifests
	for _, manifest := range testCase.Manifests {
		gomega.Eventually(func() error {
			err := s.TestInstallation.Actions.Kubectl().ApplyFile(s.Ctx, manifest)
			return err
		}, 10*time.Second, 1*time.Second).Should(gomega.Succeed(), "can apply "+manifest)
	}

	for manifest, transform := range testCase.ManifestsWithTransform {
		cur, err := os.ReadFile(manifest)
		s.Require().NoError(err)
		transformed := transform(string(cur))
		s.Require().EventuallyWithT(func(c *assert.CollectT) {
			err := s.TestInstallation.Actions.Kubectl().Apply(s.Ctx, []byte(transformed))
			assert.NoError(c, err)
		}, 10*time.Second, 1*time.Second)
	}

	// parse the expected resources and dynamic resources from the manifests, and wait until the resources are created.
	// we must wait until the resources from the manifest exist on the cluster before calling loadDynamicResources,
	// because in order to determine what dynamic resources are expected, certain resources (e.g. Gateways and
	// GatewayParameters) must already exist on the cluster.
	s.loadManifestResources(testCase)
	s.TestInstallation.Assertions.EventuallyObjectsExist(s.Ctx, testCase.manifestResources...)
	s.loadDynamicResources(testCase)
	s.TestInstallation.Assertions.EventuallyObjectsExist(s.Ctx, testCase.dynamicResources...)

	// wait until pods are ready; this assumes that pods use a well-known label
	// app.kubernetes.io/name=<name>
	allResources := slices.Concat(testCase.manifestResources, testCase.dynamicResources)
	for _, resource := range allResources {
		var ns, name string
		if pod, ok := resource.(*corev1.Pod); ok {
			ns = pod.Namespace
			name = pod.Name
		} else if deployment, ok := resource.(*appsv1.Deployment); ok {
			ns = deployment.Namespace
			name = deployment.Name
		} else {
			continue
		}
		s.TestInstallation.Assertions.EventuallyPodsRunning(s.Ctx, ns, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", defaults.WellKnownAppLabel, name),
			// Provide a longer timeout as the pod needs to be pulled and pass HCs
		}, time.Second*60, time.Second*2)
	}
}

// DeleteManifests deletes the manifests and waits until the resources are deleted.
func (s *BaseTestingSuite) DeleteManifests(testCase *TestCase) {
	// parse the expected resources and dynamic resources from the manifests (this normally would already
	// have been done via ApplyManifests, but we check again here just in case ApplyManifests was not called).
	// we need to do this before calling delete on the manifests, so we can accurately determine which dynamic
	// resources need to be deleted.
	s.loadManifestResources(testCase)
	s.loadDynamicResources(testCase)

	for _, manifest := range testCase.Manifests {
		gomega.Eventually(func() error {
			err := s.TestInstallation.Actions.Kubectl().DeleteFileSafe(s.Ctx, manifest)
			return err
		}, 10*time.Second, 1*time.Second).Should(gomega.Succeed(), "can delete "+manifest)
	}
	for manifest := range testCase.ManifestsWithTransform {
		// we don't need to transform the manifest here, as we are just deleting by filename
		gomega.Eventually(func() error {
			err := s.TestInstallation.Actions.Kubectl().DeleteFileSafe(s.Ctx, manifest)
			return err
		}, 10*time.Second, 1*time.Second).Should(gomega.Succeed(), "can delete "+manifest)
	}

	// wait until the resources are deleted
	allResources := slices.Concat(testCase.manifestResources, testCase.dynamicResources)
	s.TestInstallation.Assertions.EventuallyObjectsNotExist(s.Ctx, allResources...)

	// wait until pods created by deployments are deleted; this assumes that pods use a well-known label
	// app.kubernetes.io/name=<name>
	for _, resource := range allResources {
		if deployment, ok := resource.(*appsv1.Deployment); ok {
			s.TestInstallation.Assertions.EventuallyPodsNotExist(s.Ctx, deployment.Namespace, metav1.ListOptions{
				LabelSelector: fmt.Sprintf("%s=%s", defaults.WellKnownAppLabel, deployment.Name),
			}, time.Second*120, time.Second*2)
		}
	}
}

func (s *BaseTestingSuite) setupHelpers() {
	if s.GatewayHelper == nil {
		s.GatewayHelper = newGatewayHelper(s.TestInstallation)
	}
	if s.CrdPath == "" {
		s.CrdPath = testutils.CRDPath
	}
	var err error
	s.gvkToStructuralSchema, err = testutils.GetStructuralSchemas(filepath.Join(testutils.GitRootDirectory(), s.CrdPath))
	s.Require().NoError(err)
}

// loadManifestResources populates the `manifestResources` for the given test case, by parsing each
// manifest file into a list of resources
func (s *BaseTestingSuite) loadManifestResources(testCase *TestCase) {
	if len(testCase.manifestResources) > 0 {
		// resources have already been loaded
		return
	}

	var resources []client.Object
	for _, manifest := range testCase.Manifests {
		objs, err := testutils.LoadFromFiles(manifest, s.TestInstallation.ClusterContext.Client.Scheme(), s.gvkToStructuralSchema)
		s.Require().NoError(err)
		resources = append(resources, objs...)
	}
	for manifest := range testCase.ManifestsWithTransform {
		// we don't need to transform the resource since the transformation applies to the spec and not object metadata,
		// which ensures that parsed Go objects in manifestResources can be used normally
		objs, err := testutils.LoadFromFiles(manifest, s.TestInstallation.ClusterContext.Client.Scheme(), s.gvkToStructuralSchema)
		s.Require().NoError(err)
		resources = append(resources, objs...)
	}
	testCase.manifestResources = resources
}

// loadDynamicResources populates the `dynamicResources` for the given test case. For each Gateway
// in the test case, if it is not self-managed, then the expected dynamically provisioned resources
// are added to dynamicResources.
//
// This should only be called *after* loadManifestResources has been called and we have waited
// for all the manifest objects to be created. This is because the "is self-managed" check requires
// any dependent Gateways and GatewayParameters to exist on the cluster already.
func (s *BaseTestingSuite) loadDynamicResources(testCase *TestCase) {
	if len(testCase.dynamicResources) > 0 {
		// resources have already been loaded
		return
	}

	var dynamicResources []client.Object
	for _, obj := range testCase.manifestResources {
		if gw, ok := obj.(*gwv1.Gateway); ok {
			selfManaged, err := s.GatewayHelper.IsSelfManaged(s.Ctx, gw)
			s.Require().NoError(err)

			// if the gateway is not self-managed, then we expect a proxy deployment and service
			// to be created, so add them to the dynamic resource list
			if !selfManaged {
				proxyObjectMeta := metav1.ObjectMeta{
					Name:      gw.GetName(),
					Namespace: gw.GetNamespace(),
				}
				proxyResources := []client.Object{
					&appsv1.Deployment{ObjectMeta: proxyObjectMeta},
					&corev1.Service{ObjectMeta: proxyObjectMeta},
				}
				dynamicResources = append(dynamicResources, proxyResources...)
			}
		}
	}
	testCase.dynamicResources = dynamicResources
}

// GatewayHelper is an interface that can be implemented to provide a custom way to determine if a
// Gateway is self-managed.
type GatewayHelper interface {
	IsSelfManaged(ctx context.Context, gw *gwv1.Gateway) (bool, error)
}

type defaultGatewayHelper struct {
	gwpClient *deployerinternal.GatewayParameters
}

func newGatewayHelper(testInst *e2e.TestInstallation) *defaultGatewayHelper {
	gwpClient := deployerinternal.NewGatewayParameters(
		testInst.ClusterContext.Client,

		// empty is ok as we only care whether it's self-managed or not
		&deployer.Inputs{
			ImageInfo:                &deployer.ImageInfo{},
			GatewayClassName:         wellknown.DefaultGatewayClassName,
			WaypointGatewayClassName: wellknown.DefaultWaypointClassName,
			AgentgatewayClassName:    wellknown.DefaultAgwClassName,
		},
	)
	return &defaultGatewayHelper{gwpClient: gwpClient}
}

func (h *defaultGatewayHelper) IsSelfManaged(ctx context.Context, gw *gwv1.Gateway) (bool, error) {
	return h.gwpClient.IsSelfManaged(ctx, gw)
}

// getCurrentGatewayApiChannel returns the current Gateway API channel from the installed CRDs
func (s *BaseTestingSuite) getCurrentGatewayApiChannel() GatewayApiChannel {
	ctx := context.Background()

	// Query for Gateway CRD to detect channel from annotations
	crdList := &apiextensionsv1.CustomResourceDefinitionList{}
	err := s.TestInstallation.ClusterContext.Client.List(ctx, crdList)
	if err != nil {
		// If we can't query CRDs, default to standard (most restrictive)
		return GwApiChannelStandard
	}

	// Look for channel information in Gateway CRD annotations
	for _, crd := range crdList.Items {
		// Check if this is a Gateway API CRD
		if strings.Contains(crd.Name, "gateways.gateway.networking.k8s.io") {
			// Read channel from annotation
			if channel, exists := crd.Annotations["gateway.networking.k8s.io/channel"]; exists {
				return GatewayApiChannel(channel)
			}
		}
	}

	// Default to standard if channel annotation not found
	return GwApiChannelStandard
}

// TODO (sheidkamp) - review this method and make sure it is correct
// getCurrentGatewayApiVersion returns the current Gateway API version from the test installation
func (s *BaseTestingSuite) getCurrentGatewayApiVersion() *semver.Version {
	// Try multiple detection methods in order of preference

	// 1. Check CONFORMANCE_VERSION environment variable (set by CI/CD)
	if versionStr := os.Getenv("CONFORMANCE_VERSION"); versionStr != "" {
		if version, err := semver.NewVersion(versionStr); err == nil {
			return version
		}
	}

	// 2. Try to detect from installed CRDs by checking CRD annotations or labels
	if version := s.detectVersionFromCRDs(); version != nil {
		return version
	}

	// 3. Fallback to go module version (same logic as setup-kind.sh)
	if versionStr := s.getGoModuleVersion(); versionStr != "" {
		if version, err := semver.NewVersion(versionStr); err == nil {
			return version
		}
	}

	// 4. Ultimate fallback - return a reasonable default - assume the most restrictive version to avoid errors
	return GatewayApiVMin
}

// detectVersionFromCRDs attempts to detect the Gateway API version from installed CRDs
func (s *BaseTestingSuite) detectVersionFromCRDs() *semver.Version {
	// Query for Gateway API CRDs to detect version from annotations
	ctx := context.Background()

	// Look for Gateway CRD as a representative of Gateway API version
	crdList := &apiextensionsv1.CustomResourceDefinitionList{}
	err := s.TestInstallation.ClusterContext.Client.List(ctx, crdList)
	if err != nil {
		// If we can't query CRDs, fall back to other methods
		return nil
	}

	// Look for version information in CRD annotations
	for _, crd := range crdList.Items {
		// Check if this is a Gateway API CRD
		if strings.Contains(crd.Name, "gateways.gateway.networking.k8s.io") {
			// Try to extract version from bundle-version annotation
			// Note: channel information is available in "gateway.networking.k8s.io/channel" annotation
			if versionStr, exists := crd.Annotations["gateway.networking.k8s.io/bundle-version"]; exists {
				if version, err := semver.NewVersion(versionStr); err == nil {
					return version
				}
			}
		}
	}

	return nil
}

// getGoModuleVersion gets the Gateway API version from go.mod (same as setup-kind.sh)
func (s *BaseTestingSuite) getGoModuleVersion() string {
	// Run the same command as setup-kind.sh: go list -m sigs.k8s.io/gateway-api
	cmd := exec.Command("go", "list", "-m", "sigs.k8s.io/gateway-api")
	output, err := cmd.Output()
	if err != nil {
		// If go command fails, return empty string to fall back to default
		return ""
	}

	// Parse the output: "sigs.k8s.io/gateway-api v1.4.0"
	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) >= 2 {
		return parts[1] // Return the version part
	}

	return ""
}

// shouldSkipTest determines if a test should be skipped based on channel/version requirements.
// This is the inverse of requirementsMatch - we skip if requirements are NOT met.
func (s *BaseTestingSuite) shouldSkipTest(testCase *TestCase) bool {
	if len(testCase.GatewayApiVersion) == 0 {
		return false // No requirements = run on any channel/version
	}

	currentVersion := s.getCurrentGatewayApiVersion()
	currentChannel := s.getCurrentGatewayApiChannel()

	if currentVersion == nil {
		return false // Can't determine version, don't skip (conservative)
	}

	// Use requirementsMatch and invert the result
	return !s.requirementsMatch(testCase.GatewayApiVersion, currentChannel, currentVersion)
}
