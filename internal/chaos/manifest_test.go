package chaos

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data for manifests
const (
	testBandwidthManifest = `apiVersion: chaos-mesh.org/v1alpha1
kind: NetworkChaos
metadata:
  name: solo-chaos-network-bandwidth-${UUID}
  namespace: chaos-mesh
spec:
  action: bandwidth
  mode: all
  selector:
    namespaces:
      - ${NAMESPACE}
    expressionSelectors:
      - key: solo.hedera.com/node-name
        operator: In
        values: [${NODE_NAMES}]
    labelSelectors:
      'solo.hedera.com/type': 'network-node'
  bandwidth:
    rate: ${RATE}
    limit: ${LIMIT}
    buffer: ${BUFFER}`

	testNetemManifest = `apiVersion: chaos-mesh.org/v1alpha1
kind: NetworkChaos
metadata:
  name: solo-chaos-network-netem-800ms-${UUID}
  namespace: chaos-mesh
spec:
  action: netem
  mode: all
  selector:
    namespaces:
      - ${NAMESPACE}
    labelSelectors:
      'solo.hedera.com/type': 'network-node'
      'solo.hedera.com/latency': '800ms'
  delay:
    latency: '800ms'
    correlation: '80'
    jitter: '20ms'
  rate:
    rate: '1gbps'`

	invalidManifest = `apiVersion: invalid
kind: InvalidKind
metadata:
  name: test
spec:
  invalid: field`
)

func TestLoadManifestTemplate(t *testing.T) {
	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-manifest.yml")
	
	err := os.WriteFile(testFile, []byte(testBandwidthManifest), 0644)
	require.NoError(t, err)

	// Test successful load
	content, err := LoadManifestTemplate(testFile)
	assert.NoError(t, err)
	assert.Equal(t, testBandwidthManifest, content)

	// Test file not found
	_, err = LoadManifestTemplate("/nonexistent/file.yml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read manifest file")
}

func TestSubstituteVariables(t *testing.T) {
	vars := ManifestVariables{
		UUID:      "test-uuid-123",
		Namespace: "solo-test",
		NodeNames: "node1,node2",
		Rate:      "100mbps",
		Limit:     "1048576",
		Buffer:    "10240",
	}

	result, err := SubstituteVariables(testBandwidthManifest, vars)
	assert.NoError(t, err)

	// Verify substitutions
	assert.Contains(t, result, "solo-chaos-network-bandwidth-test-uuid-123")
	assert.Contains(t, result, "- solo-test")
	assert.Contains(t, result, "values: [node1,node2]")
	assert.Contains(t, result, "rate: 100mbps")
	assert.Contains(t, result, "limit: 1048576")
	assert.Contains(t, result, "buffer: 10240")

	// Ensure original placeholders are replaced
	assert.NotContains(t, result, "${UUID}")
	assert.NotContains(t, result, "${NAMESPACE}")
	assert.NotContains(t, result, "${NODE_NAMES}")
}

func TestParseNetworkChaosManifest(t *testing.T) {
	// Test valid bandwidth manifest
	vars := ManifestVariables{
		UUID:      "test-123",
		Namespace: "solo",
		NodeNames: "node1",
		Rate:      "1gbps",
		Limit:     "20971520",
		Buffer:    "102400",
	}

	processedContent, err := SubstituteVariables(testBandwidthManifest, vars)
	require.NoError(t, err)

	manifest, err := ParseNetworkChaosManifest(processedContent)
	assert.NoError(t, err)
	assert.NotNil(t, manifest)

	// Verify parsed structure
	assert.Equal(t, "chaos-mesh.org/v1alpha1", manifest.APIVersion)
	assert.Equal(t, "NetworkChaos", manifest.Kind)
	assert.Equal(t, "solo-chaos-network-bandwidth-test-123", manifest.Metadata.Name)
	assert.Equal(t, "chaos-mesh", manifest.Metadata.Namespace)
	assert.Equal(t, "bandwidth", manifest.Spec.Action)
	assert.Equal(t, "all", manifest.Spec.Mode)

	// Test invalid YAML
	_, err = ParseNetworkChaosManifest("invalid: yaml: content:")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse NetworkChaos manifest")
}

func TestValidateNetworkChaosManifest(t *testing.T) {
	t.Run("Valid bandwidth manifest", func(t *testing.T) {
		manifest := &NetworkChaosManifest{
			APIVersion: "chaos-mesh.org/v1alpha1",
			Kind:       "NetworkChaos",
			Metadata: ManifestMetadata{
				Name:      "test-bandwidth",
				Namespace: "chaos-mesh",
			},
			Spec: NetworkChaosSpec{
				Action: "bandwidth",
				Mode:   "all",
				Bandwidth: &BandwidthSpec{
					Rate: "1gbps",
				},
			},
		}

		err := ValidateNetworkChaosManifest(manifest)
		assert.NoError(t, err)
	})

	t.Run("Valid netem manifest", func(t *testing.T) {
		manifest := &NetworkChaosManifest{
			APIVersion: "chaos-mesh.org/v1alpha1",
			Kind:       "NetworkChaos",
			Metadata: ManifestMetadata{
				Name:      "test-netem",
				Namespace: "chaos-mesh",
			},
			Spec: NetworkChaosSpec{
				Action: "netem",
				Mode:   "all",
				Delay: &DelaySpec{
					Latency: "100ms",
				},
			},
		}

		err := ValidateNetworkChaosManifest(manifest)
		assert.NoError(t, err)
	})

	t.Run("Invalid API version", func(t *testing.T) {
		manifest := &NetworkChaosManifest{
			APIVersion: "invalid/v1",
			Kind:       "NetworkChaos",
		}

		err := ValidateNetworkChaosManifest(manifest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid apiVersion")
	})

	t.Run("Invalid kind", func(t *testing.T) {
		manifest := &NetworkChaosManifest{
			APIVersion: "chaos-mesh.org/v1alpha1",
			Kind:       "PodChaos",
		}

		err := ValidateNetworkChaosManifest(manifest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid kind")
	})

	t.Run("Missing name", func(t *testing.T) {
		manifest := &NetworkChaosManifest{
			APIVersion: "chaos-mesh.org/v1alpha1",
			Kind:       "NetworkChaos",
			Metadata: ManifestMetadata{
				Namespace: "chaos-mesh",
			},
		}

		err := ValidateNetworkChaosManifest(manifest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "metadata.name is required")
	})

	t.Run("Missing bandwidth config for bandwidth action", func(t *testing.T) {
		manifest := &NetworkChaosManifest{
			APIVersion: "chaos-mesh.org/v1alpha1",
			Kind:       "NetworkChaos",
			Metadata: ManifestMetadata{
				Name:      "test",
				Namespace: "chaos-mesh",
			},
			Spec: NetworkChaosSpec{
				Action: "bandwidth",
				Mode:   "all",
			},
		}

		err := ValidateNetworkChaosManifest(manifest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "spec.bandwidth is required for bandwidth action")
	})

	t.Run("Missing delay config for netem action", func(t *testing.T) {
		manifest := &NetworkChaosManifest{
			APIVersion: "chaos-mesh.org/v1alpha1",
			Kind:       "NetworkChaos",
			Metadata: ManifestMetadata{
				Name:      "test",
				Namespace: "chaos-mesh",
			},
			Spec: NetworkChaosSpec{
				Action: "netem",
				Mode:   "all",
			},
		}

		err := ValidateNetworkChaosManifest(manifest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "spec.delay is required for netem action")
	})
}

func TestValidatePodSelector(t *testing.T) {
	t.Run("Valid pod selector with namespaces", func(t *testing.T) {
		selector := PodSelector{
			Namespaces: []string{"solo"},
		}

		err := ValidatePodSelector(selector)
		assert.NoError(t, err)
	})

	t.Run("Valid pod selector with labels", func(t *testing.T) {
		selector := PodSelector{
			LabelSelectors: map[string]string{
				"solo.hedera.com/type": "network-node",
			},
		}

		err := ValidatePodSelector(selector)
		assert.NoError(t, err)
	})

	t.Run("Valid pod selector with expression selectors", func(t *testing.T) {
		selector := PodSelector{
			ExpressionSelectors: []ExpressionSelector{
				{
					Key:      "solo.hedera.com/node-name",
					Operator: "In",
					Values:   []string{"node1", "node2"},
				},
			},
		}

		err := ValidatePodSelector(selector)
		assert.NoError(t, err)
	})

	t.Run("Empty pod selector", func(t *testing.T) {
		selector := PodSelector{}

		err := ValidatePodSelector(selector)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pod selector must have at least one selection criteria")
	})

	t.Run("Expression selector with empty key", func(t *testing.T) {
		selector := PodSelector{
			ExpressionSelectors: []ExpressionSelector{
				{
					Key:      "",
					Operator: "In",
					Values:   []string{"node1"},
				},
			},
		}

		err := ValidatePodSelector(selector)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expression selector key cannot be empty")
	})

	t.Run("Expression selector with empty values", func(t *testing.T) {
		selector := PodSelector{
			ExpressionSelectors: []ExpressionSelector{
				{
					Key:      "test.key",
					Operator: "In",
					Values:   []string{},
				},
			},
		}

		err := ValidatePodSelector(selector)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expression selector values cannot be empty")
	})
}

func TestGenerateManifestFromTemplate(t *testing.T) {
	// Create a temporary file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test-manifest.yml")
	
	err := os.WriteFile(testFile, []byte(testBandwidthManifest), 0644)
	require.NoError(t, err)

	vars := ManifestVariables{
		UUID:      "integration-test-123",
		Namespace: "solo-integration",
		NodeNames: "node1,node2,node3",
		Rate:      "500mbps",
		Limit:     "5242880",
		Buffer:    "51200",
	}

	manifest, err := GenerateManifestFromTemplate(testFile, vars)
	assert.NoError(t, err)
	assert.NotNil(t, manifest)

	// Verify the generated manifest
	assert.Equal(t, "chaos-mesh.org/v1alpha1", manifest.APIVersion)
	assert.Equal(t, "NetworkChaos", manifest.Kind)
	assert.Equal(t, "solo-chaos-network-bandwidth-integration-test-123", manifest.Metadata.Name)
	assert.Equal(t, "bandwidth", manifest.Spec.Action)
	assert.Equal(t, "500mbps", manifest.Spec.Bandwidth.Rate)
	assert.Equal(t, 5242880, manifest.Spec.Bandwidth.Limit)
	assert.Equal(t, 51200, manifest.Spec.Bandwidth.Buffer)

	// Test with invalid template
	invalidFile := filepath.Join(tempDir, "invalid-manifest.yml")
	err = os.WriteFile(invalidFile, []byte(invalidManifest), 0644)
	require.NoError(t, err)

	_, err = GenerateManifestFromTemplate(invalidFile, vars)
	assert.Error(t, err)
}

func TestNetworkChaosManifestIntegration(t *testing.T) {
	t.Run("End-to-end bandwidth manifest generation", func(t *testing.T) {
		// Create temporary manifest file
		tempDir := t.TempDir()
		manifestFile := filepath.Join(tempDir, "bandwidth-test.yml")
		
		err := os.WriteFile(manifestFile, []byte(testBandwidthManifest), 0644)
		require.NoError(t, err)

		// Define test variables
		vars := ManifestVariables{
			UUID:      "e2e-test-456",
			Namespace: "solo-e2e",
			NodeNames: "node1",  // Use single node like in real examples
			Rate:      "2gbps",
			Limit:     "41943040",
			Buffer:    "204800",
		}

		// Generate manifest from template
		manifest, err := GenerateManifestFromTemplate(manifestFile, vars)
		require.NoError(t, err)

		// Comprehensive validation
		assert.Equal(t, "chaos-mesh.org/v1alpha1", manifest.APIVersion)
		assert.Equal(t, "NetworkChaos", manifest.Kind)
		assert.Equal(t, "solo-chaos-network-bandwidth-e2e-test-456", manifest.Metadata.Name)
		assert.Equal(t, "chaos-mesh", manifest.Metadata.Namespace)
		
		// Spec validation
		assert.Equal(t, "bandwidth", manifest.Spec.Action)
		assert.Equal(t, "all", manifest.Spec.Mode)
		
		// Selector validation
		assert.Contains(t, manifest.Spec.Selector.Namespaces, "solo-e2e")
		assert.Equal(t, "network-node", manifest.Spec.Selector.LabelSelectors["solo.hedera.com/type"])
		
		// Expression selector validation
		require.Len(t, manifest.Spec.Selector.ExpressionSelectors, 1)
		exprSelector := manifest.Spec.Selector.ExpressionSelectors[0]
		assert.Equal(t, "solo.hedera.com/node-name", exprSelector.Key)
		assert.Equal(t, "In", exprSelector.Operator)
		assert.Equal(t, []string{"node1"}, exprSelector.Values)
		
		// Bandwidth spec validation
		require.NotNil(t, manifest.Spec.Bandwidth)
		assert.Equal(t, "2gbps", manifest.Spec.Bandwidth.Rate)
		assert.Equal(t, 41943040, manifest.Spec.Bandwidth.Limit)
		assert.Equal(t, 204800, manifest.Spec.Bandwidth.Buffer)
	})

	t.Run("End-to-end netem manifest generation", func(t *testing.T) {
		// Create temporary manifest file
		tempDir := t.TempDir()
		manifestFile := filepath.Join(tempDir, "netem-test.yml")
		
		err := os.WriteFile(manifestFile, []byte(testNetemManifest), 0644)
		require.NoError(t, err)

		// Define test variables
		vars := ManifestVariables{
			UUID:      "netem-e2e-789",
			Namespace: "solo-netem",
		}

		// Generate manifest from template
		manifest, err := GenerateManifestFromTemplate(manifestFile, vars)
		require.NoError(t, err)

		// Comprehensive validation
		assert.Equal(t, "solo-chaos-network-netem-800ms-netem-e2e-789", manifest.Metadata.Name)
		assert.Equal(t, "netem", manifest.Spec.Action)
		
		// Delay spec validation
		require.NotNil(t, manifest.Spec.Delay)
		assert.Equal(t, "800ms", manifest.Spec.Delay.Latency)
		assert.Equal(t, "80", manifest.Spec.Delay.Correlation)
		assert.Equal(t, "20ms", manifest.Spec.Delay.Jitter)
		
		// Rate spec validation  
		require.NotNil(t, manifest.Spec.Rate)
		assert.Equal(t, "1gbps", manifest.Spec.Rate.Rate)
	})
}