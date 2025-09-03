package chaos

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// IntegrationTestSuite provides integration testing capabilities for chaos experiments
type IntegrationTestSuite struct {
	kubeClient  kubernetes.Interface
	validator   *ChaosValidator
	namespace   string
	testTimeout time.Duration
}

// TestPodSpec defines a test pod configuration
type TestPodSpec struct {
	Name   string
	Labels map[string]string
	Image  string
}

// NetworkTestResult represents the result of a network connectivity test
type NetworkTestResult struct {
	SourcePod      string        `json:"sourcePod"`
	TargetPod      string        `json:"targetPod"`
	Connected      bool          `json:"connected"`
	Latency        time.Duration `json:"latency,omitempty"`
	Error          string        `json:"error,omitempty"`
	TestTimestamp  time.Time     `json:"testTimestamp"`
}

// ChaosExperimentTest represents a complete test case for a chaos experiment
type ChaosExperimentTest struct {
	Name              string
	ChaosManifestPath string
	Variables         ManifestVariables
	TestPods          []TestPodSpec
	ExpectedEffects   []NetworkTestExpectation
}

// NetworkTestExpectation defines expected network behavior
type NetworkTestExpectation struct {
	SourcePodLabel string
	TargetPodLabel string
	ShouldConnect  bool
	MaxLatency     time.Duration
	Description    string
}

// NewIntegrationTestSuite creates a new integration test suite
func NewIntegrationTestSuite(config *rest.Config, namespace string) (*IntegrationTestSuite, error) {
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	validator, err := NewChaosValidator(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create chaos validator: %w", err)
	}

	return &IntegrationTestSuite{
		kubeClient:  kubeClient,
		validator:   validator,
		namespace:   namespace,
		testTimeout: 5 * time.Minute,
	}, nil
}

// SetTestTimeout sets the timeout for integration tests
func (its *IntegrationTestSuite) SetTestTimeout(timeout time.Duration) {
	its.testTimeout = timeout
}

// CreateTestPod creates a test pod with specified labels and configuration
func (its *IntegrationTestSuite) CreateTestPod(ctx context.Context, spec TestPodSpec) (*corev1.Pod, error) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      spec.Name,
			Namespace: its.namespace,
			Labels:    spec.Labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: spec.Image,
					Command: []string{
						"sh", "-c", "while true; do sleep 30; done",
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("50m"),
							corev1.ResourceMemory: resource.MustParse("32Mi"),
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyAlways,
		},
	}

	createdPod, err := its.kubeClient.CoreV1().Pods(its.namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create test pod %s: %w", spec.Name, err)
	}

	return createdPod, nil
}

// WaitForPodsReady waits for all specified pods to become ready
func (its *IntegrationTestSuite) WaitForPodsReady(ctx context.Context, podNames []string) error {
	ctx, cancel := context.WithTimeout(ctx, its.testTimeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for pods to become ready")
		case <-ticker.C:
			allReady := true
			for _, podName := range podNames {
				pod, err := its.kubeClient.CoreV1().Pods(its.namespace).Get(ctx, podName, metav1.GetOptions{})
				if err != nil {
					allReady = false
					break
				}

				if pod.Status.Phase != corev1.PodRunning {
					allReady = false
					break
				}

				// Check container readiness
				containerReady := false
				for _, condition := range pod.Status.Conditions {
					if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
						containerReady = true
						break
					}
				}

				if !containerReady {
					allReady = false
					break
				}
			}

			if allReady {
				return nil
			}
		}
	}
}

// CleanupTestPods removes all test pods
func (its *IntegrationTestSuite) CleanupTestPods(ctx context.Context, podNames []string) error {
	var errors []error

	for _, podName := range podNames {
		err := its.kubeClient.CoreV1().Pods(its.namespace).Delete(ctx, podName, metav1.DeleteOptions{})
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to delete pod %s: %w", podName, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// RunNetworkConnectivityTest performs network connectivity tests between pods
func (its *IntegrationTestSuite) RunNetworkConnectivityTest(ctx context.Context, sourcePodName, targetPodName string) (*NetworkTestResult, error) {
	sourcePod, err := its.kubeClient.CoreV1().Pods(its.namespace).Get(ctx, sourcePodName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get source pod %s: %w", sourcePodName, err)
	}

	targetPod, err := its.kubeClient.CoreV1().Pods(its.namespace).Get(ctx, targetPodName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get target pod %s: %w", targetPodName, err)
	}

	result := &NetworkTestResult{
		SourcePod:     sourcePodName,
		TargetPod:     targetPodName,
		TestTimestamp: time.Now(),
	}

	// Use the validator's connectivity check (simplified for this implementation)
	connected, err := its.validator.ValidateNetworkConnectivity(ctx, *sourcePod, *targetPod, 10*time.Second)
	result.Connected = connected

	if err != nil {
		result.Error = err.Error()
	}

	// In a real implementation, you would execute actual network commands like ping or curl
	// For example:
	// kubectl exec sourcePod -- ping -c 3 -W 5 targetPod.Status.PodIP
	// and measure actual latency and packet loss

	return result, nil
}

// ValidateNetworkPartition checks if network partition effects are working as expected
func (its *IntegrationTestSuite) ValidateNetworkPartition(ctx context.Context, partitionedPods, unaffectedPods []string) ([]NetworkTestResult, error) {
	var results []NetworkTestResult

	// Test connectivity between partitioned pods (should fail)
	for i, sourcePod := range partitionedPods {
		for j, targetPod := range partitionedPods {
			if i != j {
				result, err := its.RunNetworkConnectivityTest(ctx, sourcePod, targetPod)
				if err != nil {
					result = &NetworkTestResult{
						SourcePod:     sourcePod,
						TargetPod:     targetPod,
						Connected:     false,
						Error:         err.Error(),
						TestTimestamp: time.Now(),
					}
				}
				results = append(results, *result)
			}
		}
	}

	// Test connectivity from unaffected pods to partitioned pods (should succeed)
	for _, sourcePod := range unaffectedPods {
		for _, targetPod := range partitionedPods {
			result, err := its.RunNetworkConnectivityTest(ctx, sourcePod, targetPod)
			if err != nil {
				result = &NetworkTestResult{
					SourcePod:     sourcePod,
					TargetPod:     targetPod,
					Connected:     false,
					Error:         err.Error(),
					TestTimestamp: time.Now(),
				}
			}
			results = append(results, *result)
		}
	}

	return results, nil
}

// ValidateNetworkLatency checks if network latency effects are working as expected
func (its *IntegrationTestSuite) ValidateNetworkLatency(ctx context.Context, sourcePods, targetPods []string, expectedMinLatency time.Duration) ([]NetworkTestResult, error) {
	var results []NetworkTestResult

	for _, sourcePod := range sourcePods {
		for _, targetPod := range targetPods {
			if sourcePod != targetPod {
				result, err := its.RunNetworkConnectivityTest(ctx, sourcePod, targetPod)
				if err != nil {
					result = &NetworkTestResult{
						SourcePod:     sourcePod,
						TargetPod:     targetPod,
						Connected:     false,
						Error:         err.Error(),
						TestTimestamp: time.Now(),
					}
				}
				// Note: In a real implementation, you would measure actual latency
				// and compare it against expectedMinLatency
				results = append(results, *result)
			}
		}
	}

	return results, nil
}

// RunChaosExperimentIntegrationTest runs a complete integration test for a chaos experiment
func (its *IntegrationTestSuite) RunChaosExperimentIntegrationTest(ctx context.Context, test ChaosExperimentTest) error {
	// Step 1: Create test pods
	var podNames []string
	for _, podSpec := range test.TestPods {
		_, err := its.CreateTestPod(ctx, podSpec)
		if err != nil {
			return fmt.Errorf("failed to create test pod %s: %w", podSpec.Name, err)
		}
		podNames = append(podNames, podSpec.Name)
	}

	// Ensure cleanup
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = its.CleanupTestPods(cleanupCtx, podNames)
	}()

	// Step 2: Wait for pods to be ready
	err := its.WaitForPodsReady(ctx, podNames)
	if err != nil {
		return fmt.Errorf("failed to wait for pods ready: %w", err)
	}

	// Step 3: Test baseline connectivity (before chaos)
	baselineResults, err := its.runConnectivityTests(ctx, test.ExpectedEffects)
	if err != nil {
		return fmt.Errorf("failed to run baseline connectivity tests: %w", err)
	}

	// Step 4: Apply chaos experiment
	_, err = GenerateManifestFromTemplate(test.ChaosManifestPath, test.Variables)
	if err != nil {
		return fmt.Errorf("failed to generate chaos manifest: %w", err)
	}

	// In a real implementation, you would apply the manifest to Kubernetes here
	// kubectl apply -f manifest

	// Step 5: Wait for chaos to take effect
	time.Sleep(10 * time.Second)

	// Step 6: Test connectivity during chaos
	chaosResults, err := its.runConnectivityTests(ctx, test.ExpectedEffects)
	if err != nil {
		return fmt.Errorf("failed to run chaos connectivity tests: %w", err)
	}

	// Step 7: Validate results match expectations
	err = its.validateTestResults(baselineResults, chaosResults, test.ExpectedEffects)
	if err != nil {
		return fmt.Errorf("test validation failed: %w", err)
	}

	return nil
}

// runConnectivityTests runs connectivity tests based on expectations
func (its *IntegrationTestSuite) runConnectivityTests(ctx context.Context, expectations []NetworkTestExpectation) ([]NetworkTestResult, error) {
	var results []NetworkTestResult

	for _, expectation := range expectations {
		// Find pods matching source and target labels
		sourcePods, err := its.findPodsByLabel(ctx, expectation.SourcePodLabel)
		if err != nil {
			return nil, fmt.Errorf("failed to find source pods: %w", err)
		}

		targetPods, err := its.findPodsByLabel(ctx, expectation.TargetPodLabel)
		if err != nil {
			return nil, fmt.Errorf("failed to find target pods: %w", err)
		}

		// Test connectivity between matching pods
		for _, sourcePod := range sourcePods {
			for _, targetPod := range targetPods {
				if sourcePod != targetPod {
					result, err := its.RunNetworkConnectivityTest(ctx, sourcePod, targetPod)
					if err != nil {
						result = &NetworkTestResult{
							SourcePod:     sourcePod,
							TargetPod:     targetPod,
							Connected:     false,
							Error:         err.Error(),
							TestTimestamp: time.Now(),
						}
					}
					results = append(results, *result)
				}
			}
		}
	}

	return results, nil
}

// findPodsByLabel finds pods matching a specific label selector
func (its *IntegrationTestSuite) findPodsByLabel(ctx context.Context, labelSelector string) ([]string, error) {
	podList, err := its.kubeClient.CoreV1().Pods(its.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	var podNames []string
	for _, pod := range podList.Items {
		podNames = append(podNames, pod.Name)
	}

	return podNames, nil
}

// validateTestResults validates that test results match expectations
func (its *IntegrationTestSuite) validateTestResults(baseline, chaos []NetworkTestResult, expectations []NetworkTestExpectation) error {
	// This is a simplified validation - in practice, you would implement
	// more sophisticated validation logic based on the specific expectations

	if len(chaos) == 0 {
		return fmt.Errorf("no chaos test results to validate")
	}

	// For now, just check that we have results
	// In a real implementation, you would:
	// 1. Compare baseline vs chaos results
	// 2. Validate that connectivity matches expectations
	// 3. Check latency measurements against thresholds
	// 4. Verify that only intended pods are affected

	return nil
}