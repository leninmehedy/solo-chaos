package chaos

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildSelectionCriteriaDescription(t *testing.T) {
	validator := &ChaosValidator{}

	t.Run("Empty selector", func(t *testing.T) {
		selector := PodSelector{}
		desc := validator.buildSelectionCriteriaDescription(selector)
		assert.Equal(t, "no selection criteria", desc)
	})

	t.Run("Namespaces only", func(t *testing.T) {
		selector := PodSelector{
			Namespaces: []string{"solo", "chaos-mesh"},
		}
		desc := validator.buildSelectionCriteriaDescription(selector)
		assert.Contains(t, desc, "namespaces: [solo chaos-mesh]")
	})

	t.Run("Labels only", func(t *testing.T) {
		selector := PodSelector{
			LabelSelectors: map[string]string{
				"solo.hedera.com/type": "network-node",
				"app":                  "consensus",
			},
		}
		desc := validator.buildSelectionCriteriaDescription(selector)
		assert.Contains(t, desc, "labels:")
		assert.Contains(t, desc, "solo.hedera.com/type:network-node")
	})

	t.Run("Expression selectors", func(t *testing.T) {
		selector := PodSelector{
			ExpressionSelectors: []ExpressionSelector{
				{
					Key:      "solo.hedera.com/node-name",
					Operator: "In",
					Values:   []string{"node1", "node2"},
				},
			},
		}
		desc := validator.buildSelectionCriteriaDescription(selector)
		assert.Contains(t, desc, "expression: solo.hedera.com/node-name In [node1 node2]")
	})

	t.Run("Multiple criteria", func(t *testing.T) {
		selector := PodSelector{
			Namespaces: []string{"solo"},
			LabelSelectors: map[string]string{
				"solo.hedera.com/type": "network-node",
			},
			ExpressionSelectors: []ExpressionSelector{
				{
					Key:      "solo.hedera.com/region",
					Operator: "In",
					Values:   []string{"us", "eu"},
				},
			},
		}
		desc := validator.buildSelectionCriteriaDescription(selector)
		assert.Contains(t, desc, "namespaces:")
		assert.Contains(t, desc, "labels:")
		assert.Contains(t, desc, "expression:")
	})
}

func TestFilterPodsByLabels(t *testing.T) {
	validator := &ChaosValidator{}

	// Create test pods
	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod1",
				Labels: map[string]string{
					"solo.hedera.com/type": "network-node",
					"app":                  "consensus",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod2",
				Labels: map[string]string{
					"solo.hedera.com/type": "mirror-node",
					"app":                  "mirror",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "pod3",
				Labels: map[string]string{
					"solo.hedera.com/type": "network-node",
					"app":                  "relay",
				},
			},
		},
	}

	t.Run("No label selectors", func(t *testing.T) {
		result := validator.filterPodsByLabels(pods, nil)
		assert.Len(t, result, 3)
	})

	t.Run("Single label selector", func(t *testing.T) {
		labelSelectors := map[string]string{
			"solo.hedera.com/type": "network-node",
		}
		result := validator.filterPodsByLabels(pods, labelSelectors)
		assert.Len(t, result, 2)
		assert.Equal(t, "pod1", result[0].Name)
		assert.Equal(t, "pod3", result[1].Name)
	})

	t.Run("Multiple label selectors", func(t *testing.T) {
		labelSelectors := map[string]string{
			"solo.hedera.com/type": "network-node",
			"app":                  "consensus",
		}
		result := validator.filterPodsByLabels(pods, labelSelectors)
		assert.Len(t, result, 1)
		assert.Equal(t, "pod1", result[0].Name)
	})

	t.Run("No matching pods", func(t *testing.T) {
		labelSelectors := map[string]string{
			"nonexistent": "value",
		}
		result := validator.filterPodsByLabels(pods, labelSelectors)
		assert.Len(t, result, 0)
	})
}

func TestFilterPodsByExpressions(t *testing.T) {
	validator := &ChaosValidator{}

	// Create test pods
	pods := []corev1.Pod{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node1-pod",
				Labels: map[string]string{
					"solo.hedera.com/node-name": "node1",
					"solo.hedera.com/region":    "us",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node2-pod",
				Labels: map[string]string{
					"solo.hedera.com/node-name": "node2",
					"solo.hedera.com/region":    "eu",
				},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node3-pod",
				Labels: map[string]string{
					"solo.hedera.com/node-name": "node3",
					"solo.hedera.com/region":    "ap",
				},
			},
		},
	}

	t.Run("No expression selectors", func(t *testing.T) {
		result := validator.filterPodsByExpressions(pods, nil)
		assert.Len(t, result, 3)
	})

	t.Run("In operator", func(t *testing.T) {
		expressions := []ExpressionSelector{
			{
				Key:      "solo.hedera.com/node-name",
				Operator: "In",
				Values:   []string{"node1", "node3"},
			},
		}
		result := validator.filterPodsByExpressions(pods, expressions)
		assert.Len(t, result, 2)
		assert.Equal(t, "node1-pod", result[0].Name)
		assert.Equal(t, "node3-pod", result[1].Name)
	})

	t.Run("NotIn operator", func(t *testing.T) {
		expressions := []ExpressionSelector{
			{
				Key:      "solo.hedera.com/region",
				Operator: "NotIn",
				Values:   []string{"us"},
			},
		}
		result := validator.filterPodsByExpressions(pods, expressions)
		assert.Len(t, result, 2)
		assert.Equal(t, "node2-pod", result[0].Name)
		assert.Equal(t, "node3-pod", result[1].Name)
	})

	t.Run("Exists operator", func(t *testing.T) {
		expressions := []ExpressionSelector{
			{
				Key:      "solo.hedera.com/region",
				Operator: "Exists",
				Values:   []string{}, // Values not used for Exists
			},
		}
		result := validator.filterPodsByExpressions(pods, expressions)
		assert.Len(t, result, 3) // All pods have region label
	})

	t.Run("DoesNotExist operator", func(t *testing.T) {
		expressions := []ExpressionSelector{
			{
				Key:      "nonexistent-label",
				Operator: "DoesNotExist",
				Values:   []string{},
			},
		}
		result := validator.filterPodsByExpressions(pods, expressions)
		assert.Len(t, result, 3) // None of the pods have nonexistent-label
	})

	t.Run("Multiple expressions", func(t *testing.T) {
		expressions := []ExpressionSelector{
			{
				Key:      "solo.hedera.com/node-name",
				Operator: "In",
				Values:   []string{"node1", "node2", "node3"},
			},
			{
				Key:      "solo.hedera.com/region",
				Operator: "NotIn",
				Values:   []string{"ap"},
			},
		}
		result := validator.filterPodsByExpressions(pods, expressions)
		assert.Len(t, result, 2)
		assert.Equal(t, "node1-pod", result[0].Name)
		assert.Equal(t, "node2-pod", result[1].Name)
	})
}

func TestPodMatchesLabels(t *testing.T) {
	validator := &ChaosValidator{}

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
			Labels: map[string]string{
				"solo.hedera.com/type": "network-node",
				"app":                  "consensus",
				"version":              "v1.0.0",
			},
		},
	}

	t.Run("Exact match", func(t *testing.T) {
		labelSelectors := map[string]string{
			"solo.hedera.com/type": "network-node",
			"app":                  "consensus",
		}
		result := validator.podMatchesLabels(pod, labelSelectors)
		assert.True(t, result)
	})

	t.Run("Single label match", func(t *testing.T) {
		labelSelectors := map[string]string{
			"version": "v1.0.0",
		}
		result := validator.podMatchesLabels(pod, labelSelectors)
		assert.True(t, result)
	})

	t.Run("No match - wrong value", func(t *testing.T) {
		labelSelectors := map[string]string{
			"solo.hedera.com/type": "mirror-node",
		}
		result := validator.podMatchesLabels(pod, labelSelectors)
		assert.False(t, result)
	})

	t.Run("No match - missing label", func(t *testing.T) {
		labelSelectors := map[string]string{
			"nonexistent": "value",
		}
		result := validator.podMatchesLabels(pod, labelSelectors)
		assert.False(t, result)
	})

	t.Run("Partial match", func(t *testing.T) {
		labelSelectors := map[string]string{
			"solo.hedera.com/type": "network-node",
			"nonexistent":          "value",
		}
		result := validator.podMatchesLabels(pod, labelSelectors)
		assert.False(t, result)
	})
}

func TestPodMatchesExpression(t *testing.T) {
	validator := &ChaosValidator{}

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
			Labels: map[string]string{
				"solo.hedera.com/node-name": "node1",
				"solo.hedera.com/region":    "us",
				"app":                       "consensus",
			},
		},
	}

	t.Run("In operator - match", func(t *testing.T) {
		expr := ExpressionSelector{
			Key:      "solo.hedera.com/node-name",
			Operator: "In",
			Values:   []string{"node1", "node2"},
		}
		result := validator.podMatchesExpression(pod, expr)
		assert.True(t, result)
	})

	t.Run("In operator - no match", func(t *testing.T) {
		expr := ExpressionSelector{
			Key:      "solo.hedera.com/node-name",
			Operator: "In",
			Values:   []string{"node2", "node3"},
		}
		result := validator.podMatchesExpression(pod, expr)
		assert.False(t, result)
	})

	t.Run("NotIn operator - match", func(t *testing.T) {
		expr := ExpressionSelector{
			Key:      "solo.hedera.com/region",
			Operator: "NotIn",
			Values:   []string{"eu", "ap"},
		}
		result := validator.podMatchesExpression(pod, expr)
		assert.True(t, result)
	})

	t.Run("NotIn operator - no match", func(t *testing.T) {
		expr := ExpressionSelector{
			Key:      "solo.hedera.com/region",
			Operator: "NotIn",
			Values:   []string{"us", "eu"},
		}
		result := validator.podMatchesExpression(pod, expr)
		assert.False(t, result)
	})

	t.Run("Exists operator - match", func(t *testing.T) {
		expr := ExpressionSelector{
			Key:      "app",
			Operator: "Exists",
			Values:   []string{},
		}
		result := validator.podMatchesExpression(pod, expr)
		assert.True(t, result)
	})

	t.Run("Exists operator - no match", func(t *testing.T) {
		expr := ExpressionSelector{
			Key:      "nonexistent",
			Operator: "Exists",
			Values:   []string{},
		}
		result := validator.podMatchesExpression(pod, expr)
		assert.False(t, result)
	})

	t.Run("DoesNotExist operator - match", func(t *testing.T) {
		expr := ExpressionSelector{
			Key:      "nonexistent",
			Operator: "DoesNotExist",
			Values:   []string{},
		}
		result := validator.podMatchesExpression(pod, expr)
		assert.True(t, result)
	})

	t.Run("DoesNotExist operator - no match", func(t *testing.T) {
		expr := ExpressionSelector{
			Key:      "app",
			Operator: "DoesNotExist",
			Values:   []string{},
		}
		result := validator.podMatchesExpression(pod, expr)
		assert.False(t, result)
	})

	t.Run("Unsupported operator", func(t *testing.T) {
		expr := ExpressionSelector{
			Key:      "app",
			Operator: "UnsupportedOp",
			Values:   []string{"value"},
		}
		result := validator.podMatchesExpression(pod, expr)
		assert.False(t, result)
	})

	t.Run("Missing label for In operator", func(t *testing.T) {
		expr := ExpressionSelector{
			Key:      "missing-label",
			Operator: "In",
			Values:   []string{"value"},
		}
		result := validator.podMatchesExpression(pod, expr)
		assert.False(t, result)
	})
}

func TestValidateNetworkConnectivity(t *testing.T) {
	validator := &ChaosValidator{}
	ctx := context.Background()

	t.Run("Both pods running", func(t *testing.T) {
		sourcePod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "source-pod"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		}
		targetPod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "target-pod"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		}

		result, err := validator.ValidateNetworkConnectivity(ctx, sourcePod, targetPod, 5*time.Second)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("Source pod not running", func(t *testing.T) {
		sourcePod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "source-pod"},
			Status:     corev1.PodStatus{Phase: corev1.PodPending},
		}
		targetPod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "target-pod"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		}

		result, err := validator.ValidateNetworkConnectivity(ctx, sourcePod, targetPod, 5*time.Second)
		assert.Error(t, err)
		assert.False(t, result)
		assert.Contains(t, err.Error(), "source pod source-pod is not running")
	})

	t.Run("Target pod not running", func(t *testing.T) {
		sourcePod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "source-pod"},
			Status:     corev1.PodStatus{Phase: corev1.PodRunning},
		}
		targetPod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "target-pod"},
			Status:     corev1.PodStatus{Phase: corev1.PodFailed},
		}

		result, err := validator.ValidateNetworkConnectivity(ctx, sourcePod, targetPod, 5*time.Second)
		assert.Error(t, err)
		assert.False(t, result)
		assert.Contains(t, err.Error(), "target pod target-pod is not running")
	})
}