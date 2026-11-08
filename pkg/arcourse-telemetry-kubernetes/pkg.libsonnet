local p = import 'pkg/main.libsonnet';

p.pkg({
  source: 'https://github.com/marcbran/jsonnet-plugin-kubernetes/tree/main/pkg/arcourse-telemetry-kubernetes',
  repo: 'https://github.com/marcbran/jsonnet.git',
  branch: 'arcourse/arcourse-telemetry-kubernetes',
  path: 'arcourse/arcourse-telemetry-kubernetes',
  target: 'arcourse-telemetry-kubernetes',
  external: ['root'],
  plugins: [
    p.plugin.github('marcbran/jsonnet-plugin-time', 'v0.0.0'),
    p.plugin.github('marcbran/jsonnet-plugin-telemetry', 'v0.0.0'),
    p.plugin.github('marcbran/jsonnet-plugin-html', 'v0.0.0'),
  ],
}, |||
  Ready-made Kubernetes telemetry entities for arcourse-telemetry, mounted under
  `telemetry/kubernetes`. Each entity browses a dimension (`context`, `namespace`,
  `pod`, `container`, `pvc`, `deployment`, `statefulset`, `daemonset`, `node`) via kube-state-metrics and
  cAdvisor queries, with drilldown charts and cross-links to the matching
  `arcourse-kubernetes` resource nodes (and back). The `context`, `namespace`,
  `pod`, and `container` entities also expose `logs` views: each field of an
  entity's `logs` object becomes a logs node at `logs/<field>` (by default `all`
  and `errors`), so more views are added as more fields.

  This is an opinionated, backend-coupled convenience layer: metric drilldowns
  assume Prometheus/PromQL (kube-state-metrics + cAdvisor), and the `logs`
  selectors assume Loki/LogQL, with `cluster`/`namespace`/`container` as indexed
  labels and `pod` as structured metadata (`| pod="..."`). Each entity's queries
  are overridable fields, so a different metrics or logs backend can be swapped
  in per entity.

  Assumes a `cluster` label identifies the context and that the resource tree is
  mounted at `root.kubernetes`. Each entity is exposed as an overridable field;
  `nodeList` materializes them all through `arcourse-telemetry`'s `entities.nodeList`.
|||, {
  context: p.desc('Entity spec for the cluster/context dimension, with all/errors logs views.'),
  namespace: p.desc('Entity spec for the namespace dimension, with all/errors logs views.'),
  pod: p.desc('Entity spec for the pod dimension, with a pod state timeline drilldown and all/errors logs views.'),
  container: p.desc('Entity spec for the container dimension, with restarts, cpu/memory usage, utilisation and throttling drilldowns and all/errors logs views. `usageMatchers` selects one cAdvisor source.'),
  pvc: p.desc('Entity spec for the persistent volume claim dimension, with state, usage and inodes drilldowns.'),
  deployment: p.desc('Entity spec for the deployment dimension, with capacity, rollout and pod state drilldowns. The Stuck rollout state needs the `reason` label on kube-state-metrics conditions.'),
  statefulset: p.desc('Entity spec for the statefulset dimension, with capacity, rollout and pod state drilldowns.'),
  daemonset: p.desc('Entity spec for the daemonset dimension, with capacity, rollout and pod state drilldowns and a misscheduled pods drilldown.'),
  node: p.desc('Entity spec for the node dimension, with state, requested resources, pods and node-exporter utilisation drilldowns. `nodeExporterInfo` maps node-exporter series to nodes, by `nodename` by default.'),
  nodeList: p.desc('All entity specs materialized through `entities.nodeList`.'),
})
