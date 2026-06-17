local p = import 'pkg/main.libsonnet';

p.pkg({
  source: 'https://github.com/marcbran/jsonnet-plugin-kubernetes',
  repo: 'https://github.com/marcbran/jsonnet.git',
  branch: 'plugin/kubernetes',
  path: 'plugin/kubernetes',
  target: 'kubernetes',
}, |||
  Read-only Kubernetes API requests authenticated via kubectl contexts.

  Use `context(ctx).get(path)` to fetch resources from a cluster. The kubectl context resolves the API server URL and credentials automatically.
|||, {
  context: p.desc(|||
    Returns a context-bound object for making requests to a Kubernetes cluster.
    `ctx` is the kubectl context name.

    The returned object exposes `get(path)` which sends a GET request to the Kubernetes API server at `path`.

    On success returns parsed JSON. On failure returns a `Status` object (`kind: "Status"`).
  |||),
})
