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
  `proxy(ctx, target, path, options)` to send an HTTP request to a pod or service through the API server, or
  `contexts()` to list available kubectl contexts.
  The kubectl context resolves the API server URL and credentials automatically.

  Mutating operations are exposed as plugin actions rather than jsonnet functions, so they are not
  callable during evaluation. The host invokes them by name: `patch` and `delete`.

  `exec` and `proxy` are also available as actions, for commands or requests with side effects. They take
  the same parameters as the functions as named fields (`{ ctx, target, command }` and
  `{ ctx, target, path, options }`, with `options` optional) and return the same result as JSON.
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
  proxy: p.desc(|||
    Sends an HTTP request to a pod or service through the API server's proxy subresource, using the kubectl
    context `ctx`. Modelled on JavaScript's `fetch`.

    `target` is `{ namespace, service, port, scheme }` or `{ namespace, pod, port, scheme }`. `port` is a number
    or a port name and may be omitted. `scheme` is `http` (default) or `https`; the API server does not verify
    the target's TLS certificate.

    `path` is the request path including the query string, e.g. `/_cat/indices?format=json`.

    `options` is `{ method, headers, body }`. `method` defaults to `GET`, `body` is a string.

    Returns `{ status, statusText, headers, body }`. Header names are lowercase and repeated values are
    joined with `, `. A non-2xx status is returned as a result, not as an error; this includes errors from
    the API server itself, such as a missing pod or a service without ready endpoints. The hidden
    field `json` holds `body` parsed as JSON.

    The API server consumes the `Authorization` header, so targets that require their own `Authorization`
    header cannot be reached this way.
  |||),
})
