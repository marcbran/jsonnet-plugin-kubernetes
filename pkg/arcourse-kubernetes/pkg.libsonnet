local p = import 'pkg/main.libsonnet';

p.pkg({
  source: 'https://github.com/marcbran/jsonnet-plugin-kubernetes/tree/main/pkg/arcourse-kubernetes',
  repo: 'https://github.com/marcbran/jsonnet.git',
  branch: 'arcourse/arcourse-kubernetes',
  path: 'arcourse/arcourse-kubernetes',
  target: 'arcourse-kubernetes',
  external: ['root'],
  plugins: [
    p.plugin.github('marcbran/jsonnet-plugin-kubernetes', 'v0.9.0'),
    p.plugin.github('marcbran/jsonnet-plugin-jsonnet', 'v0.4.0'),
    p.plugin.github('marcbran/jsonnet-plugin-time', 'v0.3.1'),
  ],
}, |||
  Arcourse graph for browsing Kubernetes resources.
|||, {
  graph: p.desc(|||
    Generates list and detail nodes for the resources of the configured contexts.
  |||),
  core: p.desc(|||
    Pod tables.
  |||),
  apps: p.desc(|||
    Pod and replicaset tables plus restart and scale actions for workloads.
  |||),
  'networking.k8s.io': p.desc(|||
    External links for ingress hosts.
  |||),
})
