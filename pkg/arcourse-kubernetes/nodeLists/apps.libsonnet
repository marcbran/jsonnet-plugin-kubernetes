local a = import 'arcourse-ui/main.libsonnet';
local coreLib = import 'core.libsonnet';
local kubernetes = import '../../jsonnet-plugin-kubernetes/main.libsonnet';
local time = import 'time/main.libsonnet';
local root = import 'root';

local podSelectorTable = a.table.node + coreLib.podTableNode {
  allPods:: kubernetes.get($.context, std.format('/api/v1/namespaces/%s/pods', [std.toString($.namespace)])).items,
  data:: {
    items: [
      p
      for p in $.allPods
      if std.all([std.objectHas(p.metadata.labels, k) && p.metadata.labels[k] == $.selector[k] for k in std.objectFields($.selector)])
    ],
  },
  linkSpecs:: [
    {
      at: ['items'],
      keys: [{ path: ['metadata', 'name'] }],
      value: [{ const: 'kubernetes' }, { origin: 'context' }, { origin: 'namespace' }, { param: 'pod', path: ['metadata', 'name'] }],
    },
  ],
};

local replicaSetTableNode = {
  table+:: {
    columns: [
      { label: 'Name', path: ['metadata', 'name'] },
      { label: 'Namespace', path: ['metadata', 'namespace'] },
      {
        label: 'Revision',
        paths: [['metadata', 'annotations']],
        value(item):: std.get(std.get(item.metadata, 'annotations', {}), 'deployment.kubernetes.io/revision', ''),
      },
      {
        label: 'Desired',
        paths: [['spec', 'replicas']],
        value(item):: std.get(std.get(item, 'spec', {}), 'replicas', 1),
      },
      {
        label: 'Current',
        paths: [['status', 'replicas']],
        value(item):: std.get(std.get(item, 'status', {}), 'replicas', 0),
      },
      {
        label: 'Ready',
        paths: [['status', 'readyReplicas']],
        value(item):: std.get(std.get(item, 'status', {}), 'readyReplicas', 0),
      },
      { label: 'Created', path: ['metadata', 'creationTimestamp'] },
    ],
  },
};

local replicaSetTable = a.table.node + replicaSetTableNode {
  allReplicaSets:: kubernetes.get($.context, std.format('/apis/apps/v1/namespaces/%s/replicasets', [std.toString($.namespace)])).items,
  data:: {
    items: std.reverse(std.sort([
      rs
      for rs in $.allReplicaSets
      if std.member([o.uid for o in std.get(rs.metadata, 'ownerReferences', [])], $.ownerUid)
    ], function(rs) rs.metadata.creationTimestamp)),
  },
  linkSpecs:: [
    {
      at: ['items'],
      keys: [{ path: ['metadata', 'name'] }],
      value: [{ const: 'kubernetes' }, { origin: 'context' }, { origin: 'namespace' }, { const: 'apps' }, { param: 'replicaset', path: ['metadata', 'name'] }],
    },
  ],
};

local restartNode = a.action.node {
  resource:: error 'restartNode requires a resource',
  resourceName:: error 'restartNode requires a resourceName',
  redirectTo:: error 'restartNode requires a redirectTo',
  _summary: 'Restart %s/%s in %s' % [$.resource, $.resourceName, $.namespace],
  _action: {
    plugin: 'kubernetes',
    name: 'patch',
    data: {
      ctx: $.context,
      path: std.format('/apis/apps/v1/namespaces/%s/%s/%s', [std.toString($.namespace), $.resource, $.resourceName]),
      patch: {
        spec: {
          template: {
            metadata: {
              annotations: {
                'kubectl.kubernetes.io/restartedAt': time.format(time.now(), '2006-01-02T15:04:05Z07:00'),
              },
            },
          },
        },
      },
    },
    redirect: $.redirectTo,
  },
};

local scaleNode = a.action.node {
  resource:: error 'scaleNode requires a resource',
  resourceName:: error 'scaleNode requires a resourceName',
  redirectTo:: error 'scaleNode requires a redirectTo',
  local path = std.format('/apis/apps/v1/namespaces/%s/%s/%s', [std.toString($.namespace), $.resource, $.resourceName]),
  local replicas = std.get($._params, 'replicas', null),
  _paramSpecs: [{ name: 'replicas', type: 'number', default: null }],
  _paramInitialValues:: {
    replicas: std.get(kubernetes.neat.get($.context, path).spec, 'replicas', 1),
  },
  _summary:
    if replicas == null then 'Scale %s/%s in %s' % [$.resource, $.resourceName, $.namespace]
    else 'Scale %s/%s in %s to %s replicas' % [$.resource, $.resourceName, $.namespace, std.toString(replicas)],
  _action:
    if replicas == null then { issues: [{ param: 'replicas', message: 'is required' }] }
    else if replicas < 0 || std.floor(replicas) != replicas then { issues: [{ param: 'replicas', message: 'must be a whole number of at least 0' }] }
    else {
      plugin: 'kubernetes',
      name: 'patch',
      data: {
        ctx: $.context,
        path: path,
        patch: { spec: { replicas: replicas } },
      },
      redirect: $.redirectTo,
    },
};

local nodeList = [
  [['kubernetes', '$context', 'apps', 'replicasets'], replicaSetTableNode],
  [['kubernetes', '$context', '$namespace', 'apps', 'replicasets'], replicaSetTableNode],
  [['kubernetes', '$context', '$namespace', 'apps', '$deployment', 'pods'], podSelectorTable {
    selector: kubernetes.neat.get($.context, std.format('/apis/apps/v1/namespaces/%s/deployments/%s', [std.toString($.namespace), std.toString($.deployment)])).spec.selector.matchLabels,
  }],
  [['kubernetes', '$context', '$namespace', 'apps', '$deployment', 'replicasets'], replicaSetTable {
    ownerUid: kubernetes.neat.get($.context, std.format('/apis/apps/v1/namespaces/%s/deployments/%s', [std.toString($.namespace), std.toString($.deployment)])).metadata.uid,
  }],
  [['kubernetes', '$context', '$namespace', 'apps', '$replicaset', 'pods'], podSelectorTable {
    selector: kubernetes.neat.get($.context, std.format('/apis/apps/v1/namespaces/%s/replicasets/%s', [std.toString($.namespace), std.toString($.replicaset)])).spec.selector.matchLabels,
  }],
  [['kubernetes', '$context', '$namespace', 'apps', '$statefulset', 'pods'], podSelectorTable {
    selector: kubernetes.neat.get($.context, std.format('/apis/apps/v1/namespaces/%s/statefulsets/%s', [std.toString($.namespace), std.toString($.statefulset)])).spec.selector.matchLabels,
  }],
  [['kubernetes', '$context', '$namespace', 'apps', '$daemonset', 'pods'], podSelectorTable {
    selector: kubernetes.neat.get($.context, std.format('/apis/apps/v1/namespaces/%s/daemonsets/%s', [std.toString($.namespace), std.toString($.daemonset)])).spec.selector.matchLabels,
  }],
  [['kubernetes', '$context', '$namespace', 'apps', '$deployment', 'action', 'restart'], restartNode {
    resource:: 'deployments',
    resourceName:: $.deployment,
    redirectTo:: root.kubernetes.context($.context).namespace($.namespace).apps.deployment($.deployment).pods,
  }],
  [['kubernetes', '$context', '$namespace', 'apps', '$statefulset', 'action', 'restart'], restartNode {
    resource:: 'statefulsets',
    resourceName:: $.statefulset,
    redirectTo:: root.kubernetes.context($.context).namespace($.namespace).apps.statefulset($.statefulset).pods,
  }],
  [['kubernetes', '$context', '$namespace', 'apps', '$deployment', 'action', 'scale'], scaleNode {
    resource:: 'deployments',
    resourceName:: $.deployment,
    redirectTo:: root.kubernetes.context($.context).namespace($.namespace).apps.deployment($.deployment).pods,
  }],
  [['kubernetes', '$context', '$namespace', 'apps', '$statefulset', 'action', 'scale'], scaleNode {
    resource:: 'statefulsets',
    resourceName:: $.statefulset,
    redirectTo:: root.kubernetes.context($.context).namespace($.namespace).apps.statefulset($.statefulset).pods,
  }],
  [['kubernetes', '$context', '$namespace', 'apps', '$daemonset', 'action', 'restart'], restartNode {
    resource:: 'daemonsets',
    resourceName:: $.daemonset,
    redirectTo:: root.kubernetes.context($.context).namespace($.namespace).apps.daemonset($.daemonset).pods,
  }],
];

{
  nodeList: nodeList,
}
