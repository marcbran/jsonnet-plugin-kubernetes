local p = import 'pkg/main.libsonnet';

p.pkg({
  source: 'https://github.com/marcbran/jsonnet-plugin-kubernetes',
  repo: 'https://github.com/marcbran/jsonnet.git',
  branch: 'plugin/kubernetes',
  path: 'plugin/kubernetes',
  target: 'kubernetes',
}, |||
  Read-only Kubernetes API requests authenticated via kubectl contexts.

  Use `get(ctx, path)` to fetch resources from a cluster, or `contexts()` to list available kubectl contexts.
  The kubectl context resolves the API server URL and credentials automatically.
|||, {
  contexts: p.desc(|||
    Returns all kubectl contexts from the local kubeconfig as an array.

    Each entry contains `name`, `current`, `cluster`, `authInfo`, and `namespace`.
  |||),
  get: p.desc(|||
    Sends a GET request to the Kubernetes API server at `path` using the kubectl context `ctx`.

    On success returns parsed JSON. On failure returns a `Status` object (`kind: "Status"`).
  |||),
})
