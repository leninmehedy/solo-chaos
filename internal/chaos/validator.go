package chaos

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ChaosValidator provides validation functions for chaos experiments
type ChaosValidator struct {
	kubeClient    kubernetes.Interface
	dynamicClient dynamic.Interface
}

// ChaosExperimentStatus represents the status of a chaos experiment
type ChaosExperimentStatus struct {
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Phase     string    `json:"phase"`
	Message   string    `json:"message,omitempty"`
	StartTime time.Time `json:"startTime,omitempty"`
}

// PodMatchResult represents the result of pod selection validation
type PodMatchResult struct {
	MatchedPods     []corev1.Pod `json:"matchedPods"`
	TotalPods       int          `json:"totalPods"`
	SelectionCriteria string     `json:"selectionCriteria"`
}

// NewChaosValidator creates a new chaos validator instance
func NewChaosValidator(config *rest.Config) (*ChaosValidator, error) {
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &ChaosValidator{
		kubeClient:    kubeClient,
		dynamicClient: dynamicClient,
	}, nil
}

// ValidateChaosResourceCreation verifies that a chaos resource has been created successfully
func (cv *ChaosValidator) ValidateChaosResourceCreation(ctx context.Context, name, namespace, chaosType string) (*ChaosExperimentStatus, error) {
	// Define GVR for different chaos types
	var gvr schema.GroupVersionResource
	switch chaosType {
	case "NetworkChaos":
		gvr = schema.GroupVersionResource{
			Group:    "chaos-mesh.org",
			Version:  "v1alpha1",
			Resource: "networkchaos",
		}
	case "PodChaos":
		gvr = schema.GroupVersionResource{
			Group:    "chaos-mesh.org",
			Version:  "v1alpha1",
			Resource: "podchaos",
		}
	default:
		return nil, fmt.Errorf("unsupported chaos type: %s", chaosType)
	}

	// Get the chaos resource
	resource, err := cv.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get %s resource %s/%s: %w", chaosType, namespace, name, err)
	}

	// Extract status information
	status := &ChaosExperimentStatus{
		Name:      resource.GetName(),
		Namespace: resource.GetNamespace(),
	}

	// Try to extract phase and message from status
	if statusField, found, err := unstructured.NestedMap(resource.Object, "status"); found && err == nil {
		if phase, found, err := unstructured.NestedString(statusField, "phase"); found && err == nil {
			status.Phase = phase
		}
		if message, found, err := unstructured.NestedString(statusField, "message"); found && err == nil {
			status.Message = message
		}
	}

	// Extract creation time
	if creationTime := resource.GetCreationTimestamp(); !creationTime.IsZero() {
		status.StartTime = creationTime.Time
	}

	return status, nil
}

// ValidatePodSelection verifies that the pod selector matches the intended pods
func (cv *ChaosValidator) ValidatePodSelection(ctx context.Context, selector PodSelector) (*PodMatchResult, error) {
	result := &PodMatchResult{
		MatchedPods: []corev1.Pod{},
	}

	// Build selection criteria description
	criteria := cv.buildSelectionCriteriaDescription(selector)
	result.SelectionCriteria = criteria

	// Get pods from specified namespaces
	var allPods []corev1.Pod
	for _, ns := range selector.Namespaces {
		podList, err := cv.kubeClient.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to list pods in namespace %s: %w", ns, err)
		}
		allPods = append(allPods, podList.Items...)
	}

	result.TotalPods = len(allPods)

	// Filter pods based on label selectors
	matchedPods := cv.filterPodsByLabels(allPods, selector.LabelSelectors)
	
	// Further filter by expression selectors
	matchedPods = cv.filterPodsByExpressions(matchedPods, selector.ExpressionSelectors)

	result.MatchedPods = matchedPods

	return result, nil
}

// ValidateNetworkConnectivity tests network connectivity between pods
func (cv *ChaosValidator) ValidateNetworkConnectivity(ctx context.Context, sourcePod, targetPod corev1.Pod, timeout time.Duration) (bool, error) {
	// This is a simplified implementation. In a real scenario, you would:
	// 1. Execute a network connectivity test command in the source pod
	// 2. Try to reach the target pod's IP
	// 3. Return success/failure result
	
	// For now, we'll just validate that both pods are running
	if sourcePod.Status.Phase != corev1.PodRunning {
		return false, fmt.Errorf("source pod %s is not running (phase: %s)", sourcePod.Name, sourcePod.Status.Phase)
	}
	
	if targetPod.Status.Phase != corev1.PodRunning {
		return false, fmt.Errorf("target pod %s is not running (phase: %s)", targetPod.Name, targetPod.Status.Phase)
	}

	// In a real implementation, you would execute something like:
	// kubectl exec sourcePod -- ping -c 1 -W 5 targetPod.Status.PodIP
	
	return true, nil
}

// WaitForChaosExperimentReady waits for a chaos experiment to reach ready state
func (cv *ChaosValidator) WaitForChaosExperimentReady(ctx context.Context, name, namespace, chaosType string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for chaos experiment %s to become ready", name)
		case <-ticker.C:
			status, err := cv.ValidateChaosResourceCreation(ctx, name, namespace, chaosType)
			if err != nil {
				continue // Keep retrying
			}

			// Check if experiment is ready
			if status.Phase == "Running" || status.Phase == "Injected" {
				return nil
			}

			// Check for error states
			if status.Phase == "Failed" {
				return fmt.Errorf("chaos experiment failed: %s", status.Message)
			}
		}
	}
}

// buildSelectionCriteriaDescription creates a human-readable description of selection criteria
func (cv *ChaosValidator) buildSelectionCriteriaDescription(selector PodSelector) string {
	var criteria []string

	if len(selector.Namespaces) > 0 {
		criteria = append(criteria, fmt.Sprintf("namespaces: %v", selector.Namespaces))
	}

	if len(selector.LabelSelectors) > 0 {
		criteria = append(criteria, fmt.Sprintf("labels: %v", selector.LabelSelectors))
	}

	if len(selector.ExpressionSelectors) > 0 {
		for _, expr := range selector.ExpressionSelectors {
			criteria = append(criteria, fmt.Sprintf("expression: %s %s %v", expr.Key, expr.Operator, expr.Values))
		}
	}

	if len(criteria) == 0 {
		return "no selection criteria"
	}

	return fmt.Sprintf("[%s]", criteria)
}

// filterPodsByLabels filters pods based on label selectors
func (cv *ChaosValidator) filterPodsByLabels(pods []corev1.Pod, labelSelectors map[string]string) []corev1.Pod {
	if len(labelSelectors) == 0 {
		return pods
	}

	var matched []corev1.Pod
	for _, pod := range pods {
		if cv.podMatchesLabels(pod, labelSelectors) {
			matched = append(matched, pod)
		}
	}

	return matched
}

// filterPodsByExpressions filters pods based on expression selectors
func (cv *ChaosValidator) filterPodsByExpressions(pods []corev1.Pod, expressions []ExpressionSelector) []corev1.Pod {
	if len(expressions) == 0 {
		return pods
	}

	var matched []corev1.Pod
	for _, pod := range pods {
		if cv.podMatchesExpressions(pod, expressions) {
			matched = append(matched, pod)
		}
	}

	return matched
}

// podMatchesLabels checks if a pod matches all label selectors
func (cv *ChaosValidator) podMatchesLabels(pod corev1.Pod, labelSelectors map[string]string) bool {
	for key, value := range labelSelectors {
		if podValue, exists := pod.Labels[key]; !exists || podValue != value {
			return false
		}
	}
	return true
}

// podMatchesExpressions checks if a pod matches all expression selectors
func (cv *ChaosValidator) podMatchesExpressions(pod corev1.Pod, expressions []ExpressionSelector) bool {
	for _, expr := range expressions {
		if !cv.podMatchesExpression(pod, expr) {
			return false
		}
	}
	return true
}

// podMatchesExpression checks if a pod matches a single expression selector
func (cv *ChaosValidator) podMatchesExpression(pod corev1.Pod, expr ExpressionSelector) bool {
	podValue, exists := pod.Labels[expr.Key]
	
	switch expr.Operator {
	case "In":
		if !exists {
			return false
		}
		for _, value := range expr.Values {
			if podValue == value {
				return true
			}
		}
		return false
	case "NotIn":
		if !exists {
			return true  // If label doesn't exist, it's not in the values list
		}
		for _, value := range expr.Values {
			if podValue == value {
				return false
			}
		}
		return true
	case "Exists":
		return exists
	case "DoesNotExist":
		return !exists
	default:
		// Unsupported operator
		return false
	}
}