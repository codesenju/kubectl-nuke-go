# FAQ

## What does this tool do?

kubectl-nuke is a kubectl plugin that forcefully deletes Kubernetes resources, particularly namespaces stuck in the Terminating state. It removes finalizers when necessary to complete the deletion process.

## How is this different from regular kubectl delete?

Regular `kubectl delete namespace` can get stuck if there are finalizers preventing deletion. kubectl-nuke detects this situation and automatically removes finalizers to force the deletion to complete.

## Is it safe to use?

It is safe if you understand what you're doing. Force-removing finalizers can cause resources in the namespace to be deleted without their normal cleanup procedures. Use with caution in production environments and ensure you understand the implications.

## Can I use this in CI/CD?

Yes, the tool is designed to be scriptable and can be used in automation pipelines. It provides clear exit codes and status messages suitable for automated environments.

## How do I use the binary?

Download the appropriate `kubectl-nuke` binary for your platform from the [Releases](https://github.com/codesenju/kubectl-nuke-go/releases) page, make it executable, and run:

```sh
# Direct usage
./kubectl-nuke ns <namespace>
./kubectl-nuke namespace <namespace>

# As kubectl plugin (after installing to PATH)
kubectl nuke ns <namespace>
kubectl nuke namespace <namespace>
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

The tool waits up to 10 seconds for a normal deletion to complete. If the namespace is still in Terminating state after that, it will remove finalizers and wait up to 20 more seconds for the forced deletion to complete.

## How do I contribute?

See [CONTRIBUTING.md](../CONTRIBUTING.md) for contribution guidelines.

## How do I get help?

- Run `kubectl-nuke --help` for command help
- Run `kubectl-nuke ns --help` for namespace command help
- Check the documentation in the `docs/` folder
- Open an issue on GitHub for bugs or feature requests
