# Chaos Testing Framework

This directory contains the integration testing framework for Solo Chaos NetworkChaos experiments using Taskfile.

## Overview

The test framework simulates how a developer executes chaos testing manually on their laptop by providing automated end-to-end tests that:

1. Deploy a Solo network
2. Install Chaos Mesh
3. Apply netem chaos experiments (network latency simulation)
4. Deploy cluster diagnostics 
5. Verify network latency effects
6. Clean up the environment

## Usage

### Prerequisites

Ensure you have:
- Kubernetes cluster (local or remote)
- `kubectl` configured and connected
- Task installed (`task --version` should show 3.39.2+)
- Solo network dependencies available

### Available Tasks

List all available test tasks:
```bash
cd test/
task --list
```

### Main E2E Test

Run the complete end-to-end test for US region netem experiments:

```bash
cd test/
task e2e-test-netem-region-us
```

This executes the full workflow:
- Deploys 5-node Solo network
- Installs Chaos Mesh
- Applies network latency experiments:
  - US region: 50ms latency
  - EU region: 100ms latency  
  - AP region: 200ms latency
  - High latency node: 800ms latency
- Deploys cluster diagnostics pods
- Runs automated ping tests to verify latency
- Cleans up all resources

### Manual Testing

For interactive testing and debugging:

#### 1. Deploy cluster diagnostics only
```bash
cd test/
task deploy-cluster-diagnostics REGION=us
```

#### 2. Execute into diagnostics pod
```bash
cd test/
task exec-cluster-diagnostics NODE_NAME=node3 REGION=us
```

Once inside the pod, manually run ping tests:
```bash
# Test AP region (expected ~200ms latency)
ping node1-svc.cluster-diagnostics.svc.cluster.local

# Test EU region (expected ~100ms latency) 
ping node2-svc.cluster-diagnostics.svc.cluster.local

# Test high latency node (expected ~800ms latency)
ping node4-svc.cluster-diagnostics.svc.cluster.local

# Exit the pod
exit
```

#### 3. Run specific ping tests
```bash
cd test/
task test-ping-manual SOURCE_NODE=node3 TARGET_REGION=ap
task test-ping-manual SOURCE_NODE=node3 TARGET_REGION=eu
```

#### 4. List active experiments
```bash
cd test/
task list-chaos-experiments
```

#### 5. Clean up manually
```bash
cd test/
task cleanup-test-environment
```

### Individual Components

You can also run components separately:

#### Deploy only cluster diagnostics
```bash
cd test/
task deploy-cluster-diagnostics REGION=us
```

#### Verify latency without full e2e
```bash
cd test/
task verify-network-latency REGION=us EXPECTED_LATENCY_US="~50ms" EXPECTED_LATENCY_EU="~100ms" EXPECTED_LATENCY_AP="~200ms"
```

## How It Works

### Network Latency Simulation

The framework applies NetworkChaos experiments that simulate regional network latencies:

- **US region (node3)**: 50ms base latency
- **EU region (node2)**: 100ms base latency  
- **AP region (node1)**: 200ms base latency
- **High latency node (node4)**: 800ms latency

### Cluster Diagnostics

The cluster diagnostics deployment creates pods with networking tools (ping, iperf3, etc.) that are labeled with region information matching the Solo network nodes. This allows testing network effects between simulated regions.

### Test Validation

The framework validates that:
1. NetworkChaos experiments are successfully created
2. Cluster diagnostics pods start and are ready
3. Ping latency between regions matches expected values
4. All resources are properly cleaned up

## Extending the Framework

### Adding New Test Regions

To add tests for other regions (e.g., EU, AP), copy the `e2e-test-netem-region-us` task and modify:

```yaml
e2e-test-netem-region-eu:
  desc: End-to-end test for netem chaos experiments in EU region
  vars:
    REGION: "eu"
    EXPECTED_LATENCY_US: "~100ms"  # Different expected values
    EXPECTED_LATENCY_EU: "~50ms"   # from EU perspective
    EXPECTED_LATENCY_AP: "~200ms"
  # ... rest same as US test
```

### Adding New Experiment Types

Create new test tasks for other chaos experiments:

```yaml
e2e-test-bandwidth-limitation:
  desc: End-to-end test for bandwidth limitation experiments
  cmds:
    - task -f ../Taskfile.yml chaos:network:consensus-network-bandwidth
    # ... validation logic
```

### Custom Validation

Add custom validation tasks for specific network effects:

```yaml
verify-bandwidth-limitation:
  desc: Verify bandwidth limitation using iperf3
  cmds:
    - kubectl exec deployment/cluster-diagnostics-node3 -n cluster-diagnostics -- iperf3 -c node1-svc.cluster-diagnostics.svc.cluster.local -t 10
```

## Troubleshooting

### Common Issues

1. **Chaos Mesh not ready**: Increase timeout in wait commands
2. **Cluster diagnostics pods not starting**: Check if namespace exists and images are available
3. **Ping tests fail**: Verify NetworkChaos experiments are active with `task list-chaos-experiments`
4. **Clean up fails**: Manually delete resources with `kubectl delete`

### Debug Commands

```bash
# Check chaos mesh status
kubectl get pods -n chaos-mesh

# Check active chaos experiments  
kubectl get networkchaos -n chaos-mesh

# Check cluster diagnostics status
kubectl get pods -n cluster-diagnostics

# View chaos experiment logs
kubectl describe networkchaos -n chaos-mesh

# Debug networking issues
kubectl exec -it deployment/cluster-diagnostics-node3 -n cluster-diagnostics -- ip route
kubectl exec -it deployment/cluster-diagnostics-node3 -n cluster-diagnostics -- nslookup node1-svc.cluster-diagnostics.svc.cluster.local
```

## Integration with CI/CD

The test framework can be integrated into CI/CD pipelines:

```bash
# In your CI script
cd test/
task e2e-test-netem-region-us

# Check exit code for test result
if [ $? -eq 0 ]; then
  echo "✅ Chaos testing passed"
else
  echo "❌ Chaos testing failed"
  exit 1
fi
```

For CI environments, you may want to:
- Use shorter timeouts
- Skip cleanup to preserve debug information
- Generate test reports
- Take screenshots/logs on failures