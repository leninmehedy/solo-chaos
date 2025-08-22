# solo-chaos
A reusable sets of chaos tests for solo test network. It can be used to test the resilience and stability of the solo network by introducing various types of failures and disruptions.

## Prerequisites
- [Helm](https://helm.sh/)
- [Kubectl](https://kubernetes.io/docs/tasks/tools/)
- [k9s](https://k9scli.io/)
- [Kind](https://kind.sigs.k8s.io/)
- [Taskfile](https://taskfile.dev/)
- [solo](https://github.com/hiero/solo)

## Example usage

### Kill 1/3rd of the nodes
- Deploy a 7 nodes network using `solo`
- Run the chaos test to kill 1/3rd of the nodes:

```bash
task run:pod-chaos --nodes 7 --kill 3
```

