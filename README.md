# solo-chaos
A reusable sets of chaos tests for solo test network. It can be used to test the resilience and stability of the solo network by introducing various types of failures and disruptions.

## Project Structure

```
solo-chaos/
├── chaos/                              # Chaos experiments and taskfiles
│   ├── Taskfile.yml                    # Main chaos taskfile (renamed for independence)
│   ├── Taskfile.chaos.network.yml     # Network chaos tasks
│   ├── Taskfile.chaos.pod.yml         # Pod chaos tasks
│   ├── network/                        # Network chaos experiment configs
│   │   ├── consensus-node-bandwidth.yml
│   │   ├── netem-800ms.yml
│   │   ├── netem-ap-melbourne.yml
│   │   ├── netem-eu-london.yml
│   │   └── netem-us-ohio.yml
│   └── pod/                           # Pod chaos experiment configs
│       ├── consensus-node-failure.yml
│       └── consensus-node-kill.yml
├── dev/                               # Development tools and configs
│   ├── taskfile/                      # Task configuration files
│   └── k8s/                          # Kubernetes manifests
└── cmd/                              # Go applications
    └── hammer/                       # Chaos testing tool
```

## Prerequisites
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (macOS: ensure at least 32GB RAM and 8 CPU cores configured)
- [Helm](https://helm.sh/)
- [Kubectl](https://kubernetes.io/docs/tasks/tools/)
- [k9s](https://k9scli.io/)
- [Kind](https://kind.sigs.k8s.io/)
- [Task](https://taskfile.dev/) (install via Homebrew: `brew install go-task`)
- [solo](https://github.com/hiero/solo)
- [`jq`](https://stedolan.github.io/jq/) (install via Homebrew: `brew install jq`)

## Quick Start

### Setup
```bash
task setup 
```

### Deploy a 5 nodes network
```bash 
task deploy-network
```

### Deploy Chaos Mesh
```bash 
task install-chaos-mesh
```

## Chaos Testing

### Pod Chaos Experiments

#### Kill one of the nodes
Run the chaos test to kill one of the nodes:
```bash
task chaos:pod:consensus-pod-kill NODE_NAMES=node5
```

#### Cause pod failure
Run the chaos test to trigger pod failure for some of the nodes:
```bash
task chaos:pod:consensus-pod-failure NODE_NAMES=node5 DURATION=60s
```

### Network Chaos Experiments

#### Network bandwidth limitation
Run the chaos test to limit network bandwidth:
```bash
task chaos:network:consensus-network-bandwidth NODE_NAMES=node1 RATE=1gbps
```

#### Network latency simulation (netem)
Run network emulation chaos tests to simulate realistic global network conditions:
```bash
task chaos:network:consensus-network-netem
```

This applies comprehensive network latency emulation with proper round-trip times (RTT) between global regions:

##### Global Latency Matrix
- **us ↔ us**: 20ms RTT (10ms one-way)
- **us ↔ eu**: 100ms RTT (50ms one-way) 
- **us ↔ ap**: 200ms RTT (100ms one-way)
- **eu ↔ eu**: 20ms RTT (10ms one-way)
- **eu ↔ ap**: 300ms RTT (150ms one-way)
- **ap ↔ ap**: 20ms RTT (10ms one-way)
- **High latency test**: 800ms delay

##### Technical Implementation
Each regional configuration file contains multiple NetworkChaos resources using the `target` attribute:

- **`netem-us-ohio.yml`**: US intra-region + US→EU + US→AP latencies
- **`netem-eu-london.yml`**: EU intra-region + EU→US + EU→AP latencies
- **`netem-ap-melbourne.yml`**: AP intra-region + AP→US + AP→EU latencies
- **`netem-800ms.yml`**: High latency testing

##### NetworkChaos Resources Created
The task creates 10 NetworkChaos resources with fixed names (no UUIDs):

**US region resources:**
- `solo-chaos-network-netem-us-to-us` (10ms)
- `solo-chaos-network-netem-us-to-eu` (50ms)
- `solo-chaos-network-netem-us-to-ap` (100ms)

**EU region resources:**
- `solo-chaos-network-netem-eu-to-eu` (10ms)
- `solo-chaos-network-netem-eu-to-us` (50ms)
- `solo-chaos-network-netem-eu-to-ap` (150ms)

**AP region resources:**
- `solo-chaos-network-netem-ap-to-ap` (10ms)
- `solo-chaos-network-netem-ap-to-us` (100ms)
- `solo-chaos-network-netem-ap-to-eu` (150ms)

**Test resource:**
- `solo-chaos-network-netem-800ms` (800ms)

##### Configuration Example
```yaml
# Cross-region latency (us-to-eu)
spec:
  selector:
    labelSelectors:
      'solo.hedera.com/region': 'us'
  target:
    selector:
      labelSelectors:
        'solo.hedera.com/region': 'eu'
  delay:
    latency: '50ms'  # One-way latency for 100ms RTT
```

**Note:** Using fixed resource names ensures that subsequent runs replace existing resources rather than creating duplicates, preventing latency accumulation.

#### Cluster diagnostics for network testing
Deploy diagnostic pods to test network connectivity and analyze chaos experiment effects:

```bash
# Deploy cluster diagnostics pod (defaults to 'us' region)
task chaos:network:deploy-cluster-diagnostics

# Deploy cluster diagnostics pod with specific region
task chaos:network:deploy-cluster-diagnostics REGION=eu
task chaos:network:deploy-cluster-diagnostics REGION=ap
task chaos:network:deploy-cluster-diagnostics REGION=us

# Exec into the diagnostics pod for interactive testing
task chaos:network:exec-cluster-diagnostics

# Clean up diagnostics pod when done
task chaos:network:cleanup-cluster-diagnostics
```

The diagnostics pod includes useful network tools:
- **Connectivity testing**: `ping`, `traceroute`, `netcat`
- **Performance testing**: `iperf3` for bandwidth/latency measurement
- **Packet analysis**: `tcpdump` for network debugging
- **DNS testing**: `dig`, `nslookup` for DNS resolution
- **General utilities**: `curl`, `jq` for API testing

**Example usage inside the diagnostics pod:**
```bash
# Test connectivity to consensus nodes (update service names based on your Solo setup)
ping network-node1.solo.svc.cluster.local

# Measure latency with iperf3
iperf3 -c network-node2.solo.svc.cluster.local

# Check active NetworkChaos experiments
kubectl get networkchaos -n chaos-mesh

# Test HTTP connectivity
curl -I http://network-node3.solo.svc.cluster.local:8080
```

**Region Configuration:** The cluster-diagnostics pod is deployed in the solo namespace with a configurable `solo.hedera.com/region` label (defaults to 'us' if not specified). This allows you to test network chaos effects from different regional perspectives by deploying the diagnostics pod with the appropriate region label to match your testing scenario.

### Hammer Job Testing

#### Deploy the Hammer Job
To deploy the image, run:
```bash
task build:image
```

To deploy the solo-chaos-hammer job to your Kubernetes cluster, run:
```bash
task deploy-hammer-job 
```

Introduce faults to the network while the hammer job is running. For example, you can kill a node pod (node5) by running:
```bash
task chaos:pod:consensus-pod-kill NODE_NAMES=node5
```

## Running Chaos Tests Independently

You can run chaos tests independently from the chaos directory. When you're in the `chaos/` directory, you'll only see chaos-specific tasks:

```bash
# Navigate to chaos directory
cd chaos

# List available chaos tasks (shows only chaos tasks)
task --list

# Run specific chaos tests with simplified names
task pod:consensus-pod-kill NODE_NAMES=node5
task network:consensus-network-netem
task show-experiment-status NAME=<experiment-name> TYPE=<PodChaos|NetworkChaos>
```

**Expected output when running `task --list` from `chaos/` directory:**
```
* show-experiment-status:                    Show the status of the pod chaos experiment
* network:consensus-network-bandwidth:       Run Network Chaos experiments (limited bandwidth)
* network:consensus-network-netem:           Run Network Chaos experiments (network emulation)
* pod:consensus-pod-failure:                 Run Pod Chaos experiments (failure)
* pod:consensus-pod-kill:                    Run Pod Chaos experiments (kill)
```

## Available Tasks

Run `task --list` to see all available tasks:

### Core Tasks
- `task setup` - Initialize the environment
- `task deploy-network` - Deploy a n-node Solo network
- `task destroy-network` - Destroy the Solo network
- `task install-chaos-mesh` - Install Chaos Mesh
- `task uninstall-chaos-mesh` - Uninstall Chaos Mesh

### Pod Chaos Tasks
- `task chaos:pod:consensus-pod-kill` - Kill consensus pods
- `task chaos:pod:consensus-pod-failure` - Cause pod failures

### Network Chaos Tasks  
- `task chaos:network:consensus-network-bandwidth` - Limit network bandwidth
- `task chaos:network:consensus-network-netem` - Apply network emulation for different latencies
- `task chaos:network:deploy-cluster-diagnostics` - Deploy cluster diagnostics pod for network testing (supports REGION parameter)
- `task chaos:network:exec-cluster-diagnostics` - Exec into cluster diagnostics pod
- `task chaos:network:cleanup-cluster-diagnostics` - Remove cluster diagnostics pod

### Utility Tasks
- `task chaos:show-experiment-status` - Show chaos experiment status
- `task deploy-hammer-job` - Deploy chaos testing job
- `task destroy-hammer-job` - Remove chaos testing job