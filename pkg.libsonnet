local p = import 'pkg/main.libsonnet';

p.pkg({
  source: 'https://github.com/marcbran/jsonnet-plugin-kubernetes',
  repo: 'https://github.com/marcbran/jsonnet.git',
  branch: 'plugin/kubernetes',
  path: 'plugin/kubernetes',
  target: 'kubernetes',
}, |||
  Kubernetes API access authenticated via kubectl contexts.

  Use `get(ctx, path)` to fetch resources from a cluster, `exec(ctx, target, command)` to run a command in a pod,
  or `contexts()` to list available kubectl contexts.
  The kubectl context resolves the API server URL and credentials automatically.

  Mutating operations are exposed as plugin actions rather than jsonnet functions, so they are not
  callable during evaluation. The host invokes them by name: `patch`, `delete`, and `restart`.
|||, {
  contexts: p.desc(|||
    Returns all kubectl contexts from the local kubeconfig as an array.

    Each entry contains `name`, `current`, `cluster`, `authInfo`, and `namespace`.
  |||),
  get: p.desc(|||
    Sends a GET request to the Kubernetes API server at `path` using the kubectl context `ctx`.

    On success returns parsed JSON. On failure returns a `Status` object (`kind: "Status"`).
  |||),
  exec: p.desc(|||
    Runs `command` (an array of strings) in a pod using the kubectl context `ctx`.

    `target` is `{ namespace, pod, container }`. An omitted `container` selects the pod's default container.

    Returns `{ exitCode, stdout, stderr }`. A non-zero exit code is returned as a result, not as an error.
    The hidden field `json` holds `stdout` parsed as JSON.
  |||),
})
