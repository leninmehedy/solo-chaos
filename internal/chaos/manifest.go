package chaos

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// NetworkChaosManifest represents a Chaos Mesh NetworkChaos resource
type NetworkChaosManifest struct {
	APIVersion string                 `yaml:"apiVersion"`
	Kind       string                 `yaml:"kind"`
	Metadata   ManifestMetadata       `yaml:"metadata"`
	Spec       NetworkChaosSpec       `yaml:"spec"`
}

// ManifestMetadata represents the metadata section
type ManifestMetadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

// NetworkChaosSpec represents the spec section of NetworkChaos
type NetworkChaosSpec struct {
	Action    string                 `yaml:"action"`
	Mode      string                 `yaml:"mode"`
	Selector  PodSelector            `yaml:"selector"`
	Bandwidth *BandwidthSpec         `yaml:"bandwidth,omitempty"`
	Delay     *DelaySpec             `yaml:"delay,omitempty"`
	Rate      *RateSpec              `yaml:"rate,omitempty"`
}

// PodSelector represents the pod selector configuration
type PodSelector struct {
	Namespaces          []string          `yaml:"namespaces,omitempty"`
	LabelSelectors      map[string]string `yaml:"labelSelectors,omitempty"`
	ExpressionSelectors []ExpressionSelector `yaml:"expressionSelectors,omitempty"`
}

// ExpressionSelector represents expression-based selectors
type ExpressionSelector struct {
	Key      string   `yaml:"key"`
	Operator string   `yaml:"operator"`
	Values   []string `yaml:"values"`
}

// BandwidthSpec represents bandwidth limitation configuration
type BandwidthSpec struct {
	Rate   string `yaml:"rate"`
	Limit  int    `yaml:"limit,omitempty"`
	Buffer int    `yaml:"buffer,omitempty"`
}

// DelaySpec represents network delay configuration
type DelaySpec struct {
	Latency     string `yaml:"latency"`
	Correlation string `yaml:"correlation,omitempty"`
	Jitter      string `yaml:"jitter,omitempty"`
}

// RateSpec represents rate limiting configuration
type RateSpec struct {
	Rate string `yaml:"rate"`
}

// ManifestVariables represents template variables for manifest generation
type ManifestVariables struct {
	UUID       string
	Namespace  string
	NodeNames  string
	Rate       string
	Limit      string
	Buffer     string
	Region     string
}

// LoadManifestTemplate loads and parses a YAML manifest template
func LoadManifestTemplate(filepath string) (string, error) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to read manifest file %s: %w", filepath, err)
	}
	return string(content), nil
}

// SubstituteVariables replaces template variables in manifest content
func SubstituteVariables(content string, vars ManifestVariables) (string, error) {
	// Create a map for template substitution
	templateVars := map[string]string{
		"UUID":       vars.UUID,
		"NAMESPACE":  vars.Namespace,
		"NODE_NAMES": vars.NodeNames,
		"RATE":       vars.Rate,
		"LIMIT":      vars.Limit,
		"BUFFER":     vars.Buffer,
		"REGION":     vars.Region,
	}

	// Use simple string replacement (mimicking envsubst behavior)
	result := content
	for key, value := range templateVars {
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result, nil
}

// ParseNetworkChaosManifest parses YAML content into NetworkChaosManifest struct
func ParseNetworkChaosManifest(yamlContent string) (*NetworkChaosManifest, error) {
	var manifest NetworkChaosManifest
	err := yaml.Unmarshal([]byte(yamlContent), &manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to parse NetworkChaos manifest: %w", err)
	}
	return &manifest, nil
}

// ValidateNetworkChaosManifest validates the structure and required fields
func ValidateNetworkChaosManifest(manifest *NetworkChaosManifest) error {
	if manifest.APIVersion != "chaos-mesh.org/v1alpha1" {
		return fmt.Errorf("invalid apiVersion: expected 'chaos-mesh.org/v1alpha1', got '%s'", manifest.APIVersion)
	}

	if manifest.Kind != "NetworkChaos" {
		return fmt.Errorf("invalid kind: expected 'NetworkChaos', got '%s'", manifest.Kind)
	}

	if manifest.Metadata.Name == "" {
		return fmt.Errorf("metadata.name is required")
	}

	if manifest.Metadata.Namespace == "" {
		return fmt.Errorf("metadata.namespace is required")
	}

	if manifest.Spec.Action == "" {
		return fmt.Errorf("spec.action is required")
	}

	if manifest.Spec.Mode == "" {
		return fmt.Errorf("spec.mode is required")
	}

	// Validate action-specific configurations
	switch manifest.Spec.Action {
	case "bandwidth":
		if manifest.Spec.Bandwidth == nil {
			return fmt.Errorf("spec.bandwidth is required for bandwidth action")
		}
		if manifest.Spec.Bandwidth.Rate == "" {
			return fmt.Errorf("spec.bandwidth.rate is required")
		}
	case "netem":
		if manifest.Spec.Delay == nil {
			return fmt.Errorf("spec.delay is required for netem action")
		}
		if manifest.Spec.Delay.Latency == "" {
			return fmt.Errorf("spec.delay.latency is required")
		}
	}

	return nil
}

// ValidatePodSelector validates that the pod selector has proper configuration
func ValidatePodSelector(selector PodSelector) error {
	if len(selector.Namespaces) == 0 && len(selector.LabelSelectors) == 0 && len(selector.ExpressionSelectors) == 0 {
		return fmt.Errorf("pod selector must have at least one selection criteria")
	}

	// Validate expression selectors
	for _, expr := range selector.ExpressionSelectors {
		if expr.Key == "" {
			return fmt.Errorf("expression selector key cannot be empty")
		}
		if expr.Operator == "" {
			return fmt.Errorf("expression selector operator cannot be empty")
		}
		if len(expr.Values) == 0 {
			return fmt.Errorf("expression selector values cannot be empty")
		}
	}

	return nil
}

// GenerateManifestFromTemplate generates a complete manifest from template and variables
func GenerateManifestFromTemplate(templatePath string, vars ManifestVariables) (*NetworkChaosManifest, error) {
	// Load template
	content, err := LoadManifestTemplate(templatePath)
	if err != nil {
		return nil, err
	}

	// Substitute variables
	processedContent, err := SubstituteVariables(content, vars)
	if err != nil {
		return nil, err
	}

	// Parse manifest
	manifest, err := ParseNetworkChaosManifest(processedContent)
	if err != nil {
		return nil, err
	}

	// Validate manifest
	err = ValidateNetworkChaosManifest(manifest)
	if err != nil {
		return nil, err
	}

	return manifest, nil
}