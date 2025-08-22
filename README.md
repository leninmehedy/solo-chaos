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

### Kill 1/3rd of the nodes
- Deploy a 7 nodes network using `solo`
- Run the chaos test to kill 1/3rd of the nodes:

```bash
task run:pod-chaos --nodes 7 --kill 3
```

