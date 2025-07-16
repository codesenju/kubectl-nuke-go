# FAQ

## What does this tool do?

kubectl-nuke is a kubectl plugin that forcefully deletes Kubernetes resources, including namespaces stuck in the Terminating state and unresponsive pods. It provides both gentle and aggressive deletion modes, automatically removes finalizers when necessary, and can force-delete pods with grace period 0.

## How is this different from regular kubectl delete?

Regular `kubectl delete namespace` can get stuck if there are finalizers preventing deletion. kubectl-nuke detects this situation and automatically removes finalizers to force the deletion to complete.

## Is it safe to use?

It is safe if you understand what you're doing. Force-removing finalizers can cause resources in the namespace to be deleted without their normal cleanup procedures. Use with caution in production environments and ensure you understand the implications.

## Can I use this in CI/CD?

Yes, the tool is designed to be scriptable and can be used in automation pipelines. It provides clear exit codes and status messages suitable for automated environments.

## How do I use the binary?

Download the appropriate `kubectl-nuke` binary for your platform from the [Releases](https://github.com/codesenju/kubectl-nuke-go/releases) page, make it executable, and run:

```sh
# Namespace deletion (direct usage)
./kubectl-nuke ns <namespace>
./kubectl-nuke namespace <namespace>
./kubectl-nuke ns <namespace> --force  # Aggressive mode

# Pod force deletion (direct usage)
./kubectl-nuke pod <pod-name> -n <namespace>
./kubectl-nuke pods <pod1> <pod2> -n <namespace>

# As kubectl plugin (after installing to PATH)
kubectl nuke ns <namespace>
kubectl nuke namespace <namespace> --force
kubectl nuke pod <pod-name> -n <namespace>
kubectl nuke pods <pod1> <pod2> -n <namespace>
```

## What's the difference between 'ns' and 'namespace' commands?

There is no difference - `namespace` is just an alias for `ns`. Both commands do exactly the same thing. Use whichever you prefer.

## Can I specify a custom kubeconfig?

Yes, use the `--kubeconfig` flag:

```sh
kubectl-nuke --kubeconfig /path/to/config ns my-namespace
```

## How do I check what version I'm running?

Use the version command:

```sh
kubectl-nuke version
```

## What happens if the namespace doesn't exist?

The tool will report an error that the namespace was not found, similar to standard kubectl behavior.

## How long does it wait before force-deleting?

For namespace deletion:
- Standard mode: Waits up to 10 seconds for normal deletion, then removes finalizers and waits up to 20 more seconds
- Force mode (`--force`): Waits up to 30 seconds for complete deletion after aggressive resource cleanup

For pod deletion: Immediate termination with grace period 0 (no waiting)

## What's the difference between standard and force mode for namespaces?

- **Standard mode** (`kubectl-nuke ns <namespace>`): Attempts normal deletion first, removes finalizers only if stuck
- **Force mode** (`kubectl-nuke ns <namespace> --force`): Immediately force-deletes all resources (pods, services, deployments, etc.) with grace period 0, then deletes the namespace

## What does pod force deletion do?

Pod force deletion (`kubectl-nuke pod <pod-name> -n <namespace>`) immediately terminates pods with grace period 0, bypassing graceful shutdown. This is useful for:
- Stuck or unresponsive pods
- Pods that won't terminate normally
- Emergency cleanup situations

⚠️ **Warning**: This can cause data loss if applications don't handle sudden termination properly.

## Can I delete multiple pods at once?

Yes! You can specify multiple pod names:

```sh
kubectl-nuke pods pod1 pod2 pod3 -n my-namespace
kubectl-nuke po nginx-123 redis-456 mysql-789 -n production
```

## What aliases are supported?

- Namespace: `ns`, `namespace`
- Pod: `pod`, `pods`, `po` (matching kubectl conventions)

## When should I use force mode vs standard mode?

**Use standard mode when:**
- Normal cleanup is acceptable
- You want to respect graceful shutdown periods
- Working in production environments

**Use force mode when:**
- Namespace is completely stuck
- You need immediate cleanup (testing/development)
- Resources are unresponsive to normal deletion
- Emergency situations requiring immediate action

## How do I contribute?

See [CONTRIBUTING.md](../CONTRIBUTING.md) for contribution guidelines.

## How do I get help?

- Run `kubectl-nuke --help` for general command help
- Run `kubectl-nuke ns --help` for namespace command help
- Run `kubectl-nuke pod --help` for pod command help
- Check the documentation in the `docs/` folder
- Open an issue on GitHub for bugs or feature requests
