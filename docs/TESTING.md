# Chaos Mesh NetworkChaos Testing Strategy

This document describes the comprehensive testing strategy for Chaos Mesh NetworkChaos experiments in the solo-chaos project.

## Overview

The testing strategy includes three levels of testing:

1. **Unit Tests** - Validate YAML manifest generation and variable substitution logic
2. **Validation Scripts** - Confirm chaos resources are created and label selectors match intended pods
3. **Integration/E2E Tests** - Deploy sample network nodes, apply chaos manifests, and verify network effects

## Architecture

### Testing Components

```
internal/chaos/
├── manifest.go          # Manifest parsing and validation
├── manifest_test.go     # Unit tests for manifest functionality
├── validator.go         # Chaos resource and pod selection validation
├── validator_test.go    # Unit tests for validation logic
├── integration.go       # Integration test framework
└── integration_test.go  # Integration test examples
```

### CLI Validation Commands

The `hammer validate` command provides three subcommands:

- `hammer validate manifest` - Validate YAML manifest generation
- `hammer validate chaos` - Validate chaos experiment status
- `hammer validate network` - Validate network connectivity and pod selection

## Running Tests

### Unit Tests

Run all unit tests including the new chaos testing functionality:

```bash
# Run all unit tests
task test:unit

# Run only chaos package tests
task test:chaos

# View test reports
cat unit-test-report.md
cat chaos-test-report.md
```

### Integration Tests

Integration tests require a Kubernetes cluster with Chaos Mesh installed:

```bash
# Set up environment
export KUBECONFIG=/path/to/your/kubeconfig

# Run integration tests
task test:chaos:integration

# View integration test report
cat chaos-integration-test-report.md
```

### Manual Validation

Use the hammer CLI tool for manual validation:

```bash
# Build the hammer tool
task build

# Validate manifest generation
./bin/hammer-linux-amd64 validate manifest --manifest-path chaos/network/consensus-node-bandwidth.yml

# Validate running chaos experiment
./bin/hammer-linux-amd64 validate chaos \
  --chaos-name solo-chaos-network-bandwidth-12345 \
  --chaos-namespace chaos-mesh \
  --chaos-type NetworkChaos \
  --wait-timeout 2m

# Validate network connectivity
./bin/hammer-linux-amd64 validate network \
  --test-namespace solo \
  --kubeconfig ~/.kube/config
```

## Test Coverage

### Unit Tests

**Manifest Testing (`internal/chaos/manifest_test.go`)**:
- ✅ Template loading from files
- ✅ Variable substitution (UUID, namespace, node names, rates, etc.)
- ✅ YAML parsing and validation
- ✅ Required field validation
- ✅ Action-specific configuration validation (bandwidth vs netem)
- ✅ Pod selector validation
- ✅ End-to-end manifest generation

**Validator Testing (`internal/chaos/validator_test.go`)**:
- ✅ Pod selection criteria building
- ✅ Label-based pod filtering
- ✅ Expression-based pod filtering (In, NotIn, Exists, DoesNotExist)
- ✅ Pod label matching logic
- ✅ Network connectivity validation

### Integration Tests

**NetworkChaos Experiment Testing**:
- 🔄 Test pod deployment with region labels
- 🔄 Chaos manifest application
- 🔄 Network partition verification
- 🔄 Network latency verification
- 🔄 Bandwidth limitation verification
- 🔄 Pod selection validation

### Validation Scripts

**CLI Command Testing**:
- ✅ Manifest validation command structure
- ✅ Chaos validation command structure  
- ✅ Network validation command structure
- ✅ Required flag validation
- ✅ Default value validation

## Test Scenarios

### 1. Network Bandwidth Limitation

**Test Case**: `BandwidthLimitation_ConsensusNodes`

```yaml
Variables:
  Rate: "100mbps"
  Limit: "1048576"  # 1MB
  Buffer: "10240"   # 10KB
  NodeNames: "node1,node2"

Expected Effects:
- Nodes 1-2: Bandwidth limited to 100mbps
- Node 3: Unaffected
- Connectivity: All nodes remain connected
```

**Validation**:
- Manifest generates correctly with specified bandwidth limits
- Chaos resource is created in Kubernetes
- Label selectors match intended pods (node1, node2)
- Bandwidth limitation takes effect

### 2. Network Latency Simulation (netem)

**Test Case**: `NetworkLatency_CrossRegion`

```yaml
Scenarios:
- EU London: 100ms latency, 80% correlation, 20ms jitter
- AP Melbourne: 200ms latency, 80% correlation, 20ms jitter  
- US Ohio: 50ms latency, 80% correlation, 10ms jitter

Expected Effects:
- Cross-region pods: Added latency based on region
- Same-region pods: Minimal latency impact
- Connectivity: All pods remain connected
```

**Validation**:
- Multiple netem manifests generate correctly
- Region-based label selectors work correctly
- Latency effects are measurable
- Jitter and correlation parameters are applied

### 3. Network Partition

**Test Case**: `NetworkPartition_RegionIsolation`

```yaml
Setup:
- US nodes: 2 pods
- EU nodes: 2 pods
- Mirror nodes: 1 pod (unaffected)

Expected Effects:
- US-EU communication: Blocked or high latency
- Intra-region communication: Normal
- Mirror node: Can reach all regions
```

**Validation**:
- Partition manifests target correct pods
- Cross-region connectivity is impacted
- Intra-region connectivity remains intact
- Non-target pods are unaffected

## Environment Setup

### Prerequisites

1. **Kubernetes Cluster**: Kind, minikube, or cloud-managed cluster
2. **Chaos Mesh**: Installed via Helm
3. **kubectl**: Configured with cluster access
4. **Go 1.24+**: For running tests
5. **Task**: Build system

### Setup Commands

```bash
# Install Chaos Mesh (if not already installed)
task install-chaos-mesh

# Create test namespace
kubectl create namespace chaos-test

# Set up environment
export KUBECONFIG=/path/to/kubeconfig
export CHAOS_TEST_NAMESPACE=chaos-test
```

### Cleanup

```bash
# Remove test resources
kubectl delete namespace chaos-test

# Uninstall Chaos Mesh (if needed)
task uninstall-chaos-mesh
```

## Contributing

When adding new chaos experiments or test cases:

1. **Add unit tests** for any new manifest templates
2. **Update integration tests** with new scenarios
3. **Document test expectations** in this guide
4. **Verify CLI validation** commands work with new manifests
5. **Update CI/CD** if new test dependencies are required

## References

- [Chaos Mesh Documentation](https://chaos-mesh.org/docs/)
- [Chaos Mesh Testing Guide](https://chaos-mesh.org/docs/testing/)
- [Kubernetes Testing Best Practices](https://kubernetes.io/docs/concepts/cluster-administration/manage-deployment/#kubectl-apply)
- [Ginkgo Testing Framework](https://onsi.github.io/ginkgo/)