# solo-chaos
A reusable sets of chaos tests for solo test network. It can be used to test the resilience and stability of the solo network by introducing various types of failures and disruptions.

## Prerequisites
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) (macOS: ensure at least 32GB RAM and 8 CPU cores configured)
- [Helm](https://helm.sh/)
- [Kubectl](https://kubernetes.io/docs/tasks/tools/)
- [k9s](https://k9scli.io/)
- [Kind](https://kind.sigs.k8s.io/)
- [Task](https://taskfile.dev/) (install via Homebrew: `brew install go-task`)
- [solo](https://github.com/hiero/solo)
- [`jq`](https://stedolan.github.io/jq/) (install via Homebrew: `brew install jq`)

## Example usage

## Setup
- Setup
```bash
task setup 
```

- Deploy a 5 nodes network
```bash 
task deploy-network NODES=5
```

- Deploy Chaos Mesh
```bash 
task install-chaos-mesh
```

### Kill some of the nodes
- Run the chaos test to kill some of the nodes (node2,node1):
```bash
task chaos:pod:consensus-pod-kill NODE_NAMES=node2,node1
```

### Cause pod failure
- Run the chaos test to trigger pod failure for some of the nodes (node2,node1):
```bash
task chaos:pod:consensus-pod-failure NODE_NAMES=node2,node1 DURATION=60s
```

