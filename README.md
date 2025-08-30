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
Run network emulation chaos tests to simulate various network conditions:
```bash
task chaos:network:consensus-network-netem
```

This applies multiple network emulation scenarios:
- 800ms delay
- AP Melbourne network conditions
- EU London network conditions  
- US Ohio network conditions

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

### Utility Tasks
- `task chaos:show-experiment-status` - Show chaos experiment status
- `task deploy-hammer-job` - Deploy chaos testing job
- `task destroy-hammer-job` - Remove chaos testing job