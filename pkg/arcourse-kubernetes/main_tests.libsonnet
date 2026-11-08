local arcourseKubernetes = import './main.libsonnet';

local manifestLiteral(expr) =
  if expr.__kind__ == 'LiteralString' then expr.value
  else if expr.__kind__ == 'Array' then [manifestLiteral(e.expr) for e in expr.elements]
  else if expr.__kind__ == 'Object' then { [f.id]: manifestLiteral(f.expr2) for f in expr.fields }
  else error 'unexpected kind ' + expr.__kind__;

local unwrap(node) = if node.__kind__ == 'Local' then unwrap(node.body) else node;

local nodePath(entry) = [part.expr.value for part in entry.expr.elements[0].expr.elements];
local isApiResources(entry) = nodePath(entry)[std.length(nodePath(entry)) - 1] == 'apiResources';
local resourceEntries(graph) = [e for e in unwrap(graph._view.jsonnet).elements if !isApiResources(e)];

local nodes(groups, globalLinkSpecs=[], hiddenGroups=[]) =
  resourceEntries(arcourseKubernetes.graph {
    manifest: false,
    hiddenGroups: hiddenGroups,
    globalLinkSpecs: globalLinkSpecs,
    data+: { groups: groups },
  });
local nodeView(entry) = entry.expr.elements[1].expr.left;
local nodeBody(entry) = entry.expr.elements[1].expr.right;
local nodeKind(entry) = [nodeView(entry).id, nodeView(entry).target.id, nodeView(entry).target.target.id];
local fieldNames(body) = [f.id for f in body.fields];
local field(body, name) = [f for f in body.fields if f.id == name][0].expr2;
local listLiteral(body) =
  local fieldName(f) = if f.id != null then f.id else f.expr1.value;
  local memberName(expr) = (if expr.target.__kind__ == 'Dollar' then '$' else expr.target.id) + '.' + expr.id;
  local literal(expr) =
    if expr.__kind__ == 'LiteralNumber' then expr.originalString
    else if expr.__kind__ == 'Object' then { [fieldName(f)]: literal(f.expr2) for f in expr.fields }
    else if expr.__kind__ == 'Index' then memberName(expr)
    else if expr.__kind__ == 'Apply' then { [memberName(expr.target)]: [literal(arg.expr) for arg in expr.arguments.positional] }
    else manifestLiteral(expr);
  literal([f for f in body.fields if f.id == 'list'][0].expr2);
local groupLists(groups, favoriteGroups=[], hiddenGroups=[], hiddenResources={}, namespaces={}) =
  local namespaceResource = { name: 'namespaces', kind: 'Namespace', namespaced: false, verbs: ['get'], version: 'v1' };
  local widgets = { name: 'widgets', kind: 'Widget', namespaced: false, verbs: ['list'], version: 'v1' };
  local entries = unwrap(arcourseKubernetes.graph {
    manifest: false,
    favoriteGroups: favoriteGroups,
    hiddenResources: hiddenResources,
    hiddenGroups: hiddenGroups,
    namespaces: namespaces,
    data+: { groups: [{ group: '', resources: [namespaceResource] }] + [{ group: g, resources: [widgets] } for g in groups] },
  }._view.jsonnet).elements;
  local body(path) = [nodeBody(e) for e in entries if nodePath(e) == path][0];
  {
    context: listLiteral(entries[1].expr.elements[1].expr),
    namespace: listLiteral(body(['kubernetes', '$context', '$namespace'])),
  };
local applyTargetChain(apply) = [apply.target.id, apply.target.target.id, apply.target.target.target.id];

{
  output(input):: input(),
  tests: [
    {
      name: 'exports the graph and the core, apps and networking.k8s.io node lists',
      input:: function() std.objectFields(arcourseKubernetes),
      expected: ['apps', 'core', 'graph', 'networking.k8s.io'],
    },
    {
      name: 'the core pod table node orders columns like kubectl and computes ready and restarts',
      input:: function()
        local a = import 'arcourse-ui/main.libsonnet';
        local node = a.table.node + arcourseKubernetes.core.podTableNode {
          data: {
            items: [{
              metadata: { name: 'api', namespace: 'default', creationTimestamp: '2026-10-12T08:14:03Z' },
              spec: { nodeName: 'node-a', containers: [{}, {}] },
              status: { phase: 'Running', podIP: '10.0.0.1', containerStatuses: [{ ready: true, restartCount: 1 }, { ready: false, restartCount: 3 }] },
            }],
          },
        };
        {
          labels: [col.label for col in node.table.columns],
          items: node.items,
        },
      expected: {
        labels: ['Name', 'Namespace', 'Ready', 'Status', 'Restarts', 'Node', 'Pod IP', 'Created'],
        items: [{ name: 'api', namespace: 'default', ready: '1/2', status: 'Running', restarts: 4, node: 'node-a', podIP: '10.0.0.1', created: '2026-10-12T08:14:03Z' }],
      },
    },
    {
      name: 'empty groups yields only the contexts and $context scaffold nodes',
      input:: function()
        local entries = nodes([]);
        { paths: [nodePath(e) for e in entries] },
      expected: {
        paths: [['kubernetes', 'contexts'], ['kubernetes', '$context']],
      },
    },
    {
      name: 'contexts node uses a.list.node, keeps the call in response and computes per-context links',
      input:: function()
        local entries = nodes([]);
        local contexts = entries[0];
        {
          kind: nodeKind(contexts),
          fieldNames: fieldNames(nodeBody(contexts)),
        },
      expected: {
        kind: ['node', 'list', 'a'],
        fieldNames: ['response', 'data', 'links'],
      },
    },
    {
      name: 'a cluster-scoped resource with list and get verbs generates a list node (a.list.node, no columns) with hidden data and a resource detail node with visible data',
      input:: function()
        local resource = { name: 'widgets', kind: 'Widget', namespaced: false, verbs: ['list', 'get'], version: 'v1' };
        local entries = nodes([{ group: '', resources: [resource] }]);
        local dataHide(entry) = [f for f in nodeBody(entry).fields if f.id == 'data'][0].Hide;
        {
          paths: [nodePath(e) for e in entries[2:]],
          listKind: nodeKind(entries[2]),
          detailKind: nodeKind(entries[3]),
          listDataHide: dataHide(entries[2]),
          detailDataHide: dataHide(entries[3]),
        },
      expected: {
        paths: [['kubernetes', '$context', 'widgets'], ['kubernetes', '$context', '$widget']],
        listKind: ['node', 'list', 'a'],
        detailKind: ['node', 'resource', 'a'],
        listDataHide: 0,
        detailDataHide: 1,
      },
    },
    {
      name: 'a cluster-scoped resource list and detail node both fetch data through kubernetes.neat.get',
      input:: function()
        local resource = { name: 'widgets', kind: 'Widget', namespaced: false, verbs: ['list', 'get'], version: 'v1' };
        local entries = nodes([{ group: '', resources: [resource] }]);
        {
          listDataChain: applyTargetChain(field(nodeBody(entries[2]), 'response')),
          detailDataChain: applyTargetChain(field(nodeBody(entries[3]), 'response')),
        },
      expected: {
        listDataChain: ['get', 'neat', 'kubernetes'],
        detailDataChain: ['get', 'neat', 'kubernetes'],
      },
    },
    {
      name: 'a resource with columns generates a table node instead of a list node',
      input:: function()
        local resource = {
          name: 'widgets',
          kind: 'Widget',
          namespaced: false,
          verbs: ['list'],
          version: 'v1',
          columns: [{ label: 'Name', path: ['metadata', 'name'] }],
        };
        local entries = nodes([{ group: '', resources: [resource] }]);
        local body = nodeBody(entries[2]);
        {
          kind: nodeKind(entries[2]),
          fieldNames: fieldNames(body),
          dataHide: [f for f in body.fields if f.id == 'data'][0].Hide,
          columnLabels: [manifestLiteral(field(col.expr, 'label')) for col in field(field(body, 'table'), 'columns').elements],
        },
      expected: {
        kind: ['node', 'table', 'a'],
        fieldNames: ['response', 'data', 'linkSpecs', 'table'],
        dataHide: 0,
        columnLabels: ['Name'],
      },
    },
    {
      name: 'a resource with only the list verb generates no detail node',
      input:: function()
        local resource = { name: 'widgets', kind: 'Widget', namespaced: false, verbs: ['list'], version: 'v1' };
        local entries = nodes([{ group: '', resources: [resource] }]);
        { paths: [nodePath(e) for e in entries[2:]] },
      expected: {
        paths: [['kubernetes', '$context', 'widgets']],
      },
    },
    {
      name: 'a resource with only the get verb generates no list node',
      input:: function()
        local resource = { name: 'widgets', kind: 'Widget', namespaced: false, verbs: ['get'], version: 'v1' };
        local entries = nodes([{ group: '', resources: [resource] }]);
        { paths: [nodePath(e) for e in entries[2:]] },
      expected: {
        paths: [['kubernetes', '$context', '$widget']],
      },
    },
    {
      name: 'a namespaced resource generates an all-namespaces list, a per-namespace list, and a detail node, each with the right linkSpec value segments',
      input:: function()
        local resource = { name: 'widgets', kind: 'Widget', namespaced: true, verbs: ['list', 'get'], version: 'v1' };
        local entries = nodes([{ group: '', resources: [resource] }]);
        local allNs = entries[2];
        local oneNs = entries[3];
        local detail = entries[4];
        {
          paths: [nodePath(e) for e in [allNs, oneNs, detail]],
          allNsLinkValue: manifestLiteral(field(nodeBody(allNs), 'linkSpecs'))[0].value,
          oneNsLinkValue: manifestLiteral(field(nodeBody(oneNs), 'linkSpecs'))[0].value,
          detailKind: nodeKind(detail),
          allNsDataChain: applyTargetChain(field(nodeBody(allNs), 'response')),
          oneNsDataChain: applyTargetChain(field(nodeBody(oneNs), 'response')),
          detailDataChain: applyTargetChain(field(nodeBody(detail), 'response')),
        },
      expected: {
        paths: [
          ['kubernetes', '$context', 'widgets'],
          ['kubernetes', '$context', '$namespace', 'widgets'],
          ['kubernetes', '$context', '$namespace', '$widget'],
        ],
        allNsLinkValue: [
          { const: 'kubernetes' },
          { origin: 'context' },
          { param: 'namespace', path: ['metadata', 'namespace'] },
          { param: 'widget', path: ['metadata', 'name'] },
        ],
        oneNsLinkValue: [
          { const: 'kubernetes' },
          { origin: 'context' },
          { origin: 'namespace' },
          { param: 'widget', path: ['metadata', 'name'] },
        ],
        detailKind: ['node', 'resource', 'a'],
        allNsDataChain: ['get', 'neat', 'kubernetes'],
        oneNsDataChain: ['get', 'neat', 'kubernetes'],
        detailDataChain: ['get', 'neat', 'kubernetes'],
      },
    },
    {
      name: 'a group gets no group node of its own, only its resource nodes, also when it is hidden',
      input:: function()
        local resource = { name: 'widgets', kind: 'Widget', namespaced: true, verbs: ['list'], version: 'v1' };
        local entries = nodes([{ group: 'acme.io', resources: [resource] }], hiddenGroups=['acme.io']);
        { paths: [nodePath(e) for e in entries[2:]] },
      expected: {
        paths: [
          ['kubernetes', '$context', 'acme.io', 'widgets'],
          ['kubernetes', '$context', '$namespace', 'acme.io', 'widgets'],
        ],
      },
    },
    {
      name: 'the context node gets the list column count and the namespaces detail node is stacked with the same columns',
      input:: function()
        local namespaces = { name: 'namespaces', kind: 'Namespace', namespaced: false, verbs: ['get'], version: 'v1' };
        local entriesWith(listColumns) =
          unwrap(arcourseKubernetes.graph {
            manifest: false,
            listColumns: listColumns,
            data+: { groups: [{ group: '', resources: [namespaces] }] },
          }._view.jsonnet).elements;
        local entry(entries, path) = [e for e in entries if nodePath(e) == path][0];
        local columns(body) = field(field(body, 'list'), 'columns').originalString;
        local summary(listColumns) =
          local entries = entriesWith(listColumns);
          local detail = nodeBody(entry(entries, ['kubernetes', '$context', '$namespace']));
          {
            context: columns(entry(entries, ['kubernetes', '$context']).expr.elements[1].expr),
            detailStacked: field(field(detail, 'resource'), 'stacked').__kind__,
            detailColumns: columns(detail),
          };
        { default: summary(4), overridden: summary(3) },
      expected: {
        default: { context: '4', detailStacked: 'LiteralBoolean', detailColumns: '4' },
        overridden: { context: '3', detailStacked: 'LiteralBoolean', detailColumns: '3' },
      },
    },
    {
      name: 'by default the context node lists the groups without dots, then the domain trees, all expanded',
      input:: function()
        local lists = groupLists(['traefik.io', 'hub.traefik.io', 'apps', 'networking.k8s.io', 'cilium.io', 'flowcontrol.apiserver.k8s.io', 'scheduling.k8s.io']);
        lists.context,
      expected: {
        columns: '4',
        groupOrder: ['apps', 'cilium.io', 'flowcontrol.apiserver.k8s.io', 'networking.k8s.io', 'scheduling.k8s.io', 'traefik.io', 'hub.traefik.io'],
      },
    },
    {
      name: 'favoriteGroups rank each group by its most specific matching favorite and keep the default order within a rank',
      input:: function()
        local lists = groupLists(
          ['apps', 'batch', 'networking.k8s.io', 'storage.k8s.io', 'coordination.k8s.io', 'cert-manager.io', 'acme.cert-manager.io', 'cilium.io'],
          favoriteGroups=['k8s.io', 'cert-manager.io', 'networking.k8s.io', 'batch'],
        );
        lists.context.groupOrder,
      expected: ['coordination.k8s.io', 'storage.k8s.io', 'cert-manager.io', 'acme.cert-manager.io', 'networking.k8s.io', 'batch', 'apps', 'cilium.io'],
    },
    {
      name: 'hiddenGroups leave groups out of the group order and list them as hidden groups on the context and namespace nodes',
      input:: function()
        local lists = groupLists(['apps', 'networking.k8s.io', 'storage.k8s.io', 'cilium.io'], hiddenGroups=['k8s.io', 'cilium.io']);
        { context: lists.context, namespace: lists.namespace },
      expected: {
        local list = {
          columns: '4',
          groupOrder: ['apps'],
          hiddenGroups: ['cilium.io', 'networking.k8s.io', 'storage.k8s.io'],
        },
        context: list,
        namespace: list,
      },
    },
    {
      name: 'the more specific of the matching favorite and hidden patterns wins and a tie goes to the favorite',
      input:: function()
        local hidden(favoriteGroups, hiddenGroups) =
          std.get(groupLists(['networking.k8s.io', 'storage.k8s.io'], favoriteGroups, hiddenGroups=hiddenGroups).context, 'hiddenGroups', null);
        {
          favoriteSubdomain: hidden(['networking.k8s.io'], ['k8s.io']),
          hiddenSubdomain: hidden(['k8s.io'], ['networking.k8s.io']),
          tie: hidden(['k8s.io'], ['k8s.io']),
        },
      expected: {
        favoriteSubdomain: ['storage.k8s.io'],
        hiddenSubdomain: ['networking.k8s.io'],
        tie: null,
      },
    },
    {
      name: 'a *.domain pattern matches only the subdomains and is more specific than the domain but less than an exact subdomain',
      input:: function()
        local lists = groupLists(
          ['cert-manager.io', 'acme.cert-manager.io', 'other.cert-manager.io', 'external-secrets.io', 'generators.external-secrets.io'],
          favoriteGroups=['cert-manager.io', 'external-secrets.io', 'other.cert-manager.io'],
          hiddenGroups=['*.cert-manager.io', '*.external-secrets.io'],
        );
        lists.context,
      expected: {
        columns: '4',
        groupOrder: ['cert-manager.io', 'external-secrets.io', 'other.cert-manager.io'],
        hiddenGroups: ['acme.cert-manager.io', 'generators.external-secrets.io'],
      },
    },
    {
      name: 'the * hidden pattern hides every group except the favorites',
      input:: function()
        local lists = groupLists(['apps', 'batch', 'cilium.io'], favoriteGroups=['batch'], hiddenGroups=['*']);
        lists.context,
      expected: {
        columns: '4',
        groupOrder: ['batch'],
        hiddenGroups: ['apps', 'cilium.io'],
      },
    },
    {
      name: 'hiddenResources keyed by group become the hidden list items of the context and namespace nodes',
      input:: function()
        local lists = groupLists(['apps'], hiddenResources={ '': ['podtemplates'], apps: ['controllerrevisions'] });
        { context: lists.context.hiddenItems, namespace: lists.namespace.hiddenItems },
      expected: {
        local hidden = { '': ['podtemplates'], apps: ['controllerrevisions'] },
        context: hidden,
        namespace: hidden,
      },
    },
    {
      name: 'namespaces overrides replace single settings per namespace on the namespace node, which picks its list by $.namespace, while the context node keeps the defaults',
      input:: function()
        local lists = groupLists(
          ['apps', 'batch', 'cilium.io'],
          favoriteGroups=['batch'],
          hiddenGroups=['cilium.io'],
          hiddenResources={ '': ['podtemplates'] },
          namespaces={
            monitoring: { favoriteGroups:: ['cilium.io'], hiddenGroups:: [] },
            'cert-manager': { listColumns:: 2, hiddenResources:: {} },
          },
        );
        lists,
      expected: {
        local default = {
          columns: '4',
          groupOrder: ['batch', 'apps'],
          hiddenGroups: ['cilium.io'],
          hiddenItems: { '': ['podtemplates'] },
        },
        context: default,
        namespace: {
          'std.get': [
            {
              'cert-manager': { columns: '2', groupOrder: ['batch', 'apps'], hiddenGroups: ['cilium.io'] },
              monitoring: { columns: '4', groupOrder: ['cilium.io', 'apps', 'batch'], hiddenItems: { '': ['podtemplates'] } },
            },
            '$.namespace',
            default,
          ],
        },
      },
    },
    {
      name: 'the context and namespace apiResources nodes link every listable resource of their scope grouped by API group, ignoring hiddenGroups and hiddenResources',
      input:: function()
        local resource(name, kind, namespaced) = { name: name, kind: kind, namespaced: namespaced, verbs: ['list', 'get'], version: 'v1' };
        local entries = unwrap(arcourseKubernetes.graph {
          manifest: false,
          favoriteGroups: ['acme.io'],
          hiddenGroups: ['apps'],
          hiddenResources: { '': ['pods'] },
          data+: { groups: [
            { group: '', resources: [resource('pods', 'Pod', true), resource('nodes', 'Node', false)] },
            { group: 'apps', resources: [resource('deployments', 'Deployment', true)] },
            { group: 'acme.io', resources: [resource('widgets', 'Widget', false)] },
          ] },
        }._view.jsonnet).elements;
        local code(expr) =
          if expr.__kind__ == 'Var' then expr.id
          else if expr.__kind__ == 'Dollar' then '$'
          else if expr.__kind__ == 'Apply' then code(expr.target) + '(' + std.join(', ', [code(arg.expr) for arg in expr.arguments.positional]) + ')'
          else if std.objectHas(expr, 'id') then code(expr.target) + '.' + expr.id
          else code(expr.target) + "['" + expr.index.value + "']";
        local fieldName(f) = if f.id != null then f.id else f.expr1.value;
        local links(expr) = { [fieldName(f)]: if f.expr2.__kind__ == 'Object' then links(f.expr2) else code(f.expr2) for f in expr.fields };
        local apiResources(path) =
          local entry = [e for e in entries if nodePath(e) == path][0];
          local body = nodeBody(entry);
          {
            view: nodeKind(entry),
            scope: code([f for f in body.fields if f.kind == 4][0].expr2),
            list: listLiteral(body),
            links: links(field(body, 'links')),
          };
        {
          context: apiResources(['kubernetes', '$context', 'apiResources']),
          namespace: apiResources(['kubernetes', '$context', '$namespace', 'apiResources']),
        },
      expected: {
        local list = { columns: '4', groupOrder: ['acme.io', 'apps'] },
        context: {
          view: ['node', 'list', 'a'],
          scope: 'root.kubernetes.context($.context)',
          list: list,
          links: {
            pods: 'scope.pods',
            nodes: 'scope.nodes',
            'acme.io': { widgets: "scope['acme.io'].widgets" },
            apps: { deployments: 'scope.apps.deployments' },
          },
        },
        namespace: {
          view: ['node', 'list', 'a'],
          scope: 'root.kubernetes.context($.context).namespace($.namespace)',
          list: list,
          links: {
            pods: 'scope.pods',
            apps: { deployments: 'scope.apps.deployments' },
          },
        },
      },
    },
    {
      name: 'a bundled specs.json entry for a core resource derives a namespaced resource with list and get verbs from its OpenAPI paths',
      input:: function()
        local spec = {
          paths: {
            '/api/v1/namespaces/{namespace}/widgets': { get: { 'x-kubernetes-group-version-kind': { kind: 'Widget' } } },
            '/api/v1/namespaces/{namespace}/widgets/{name}': { get: {} },
          },
        };
        local entries = resourceEntries(arcourseKubernetes.graph {
          manifest: false,
          contexts: [],
          specs: { 'api/v1': spec },
        });
        {
          paths: [nodePath(e) for e in entries[2:]],
        },
      expected: {
        paths: [
          ['kubernetes', '$context', 'widgets'],
          ['kubernetes', '$context', '$namespace', 'widgets'],
          ['kubernetes', '$context', '$namespace', '$widget'],
        ],
      },
    },
    {
      name: 'a bundled specs.json entry for a group resource derives the group and version from its key',
      input:: function()
        local spec = {
          paths: {
            '/apis/acme.io/v1/widgets': { get: { 'x-kubernetes-group-version-kind': { kind: 'Widget' } } },
            '/apis/acme.io/v1/widgets/{name}': { get: {} },
          },
        };
        local entries = resourceEntries(arcourseKubernetes.graph {
          manifest: false,
          contexts: [],
          specs: { 'apis/acme.io/v1': spec },
        });
        { paths: [nodePath(e) for e in entries[2:]] },
      expected: {
        paths: [
          ['kubernetes', '$context', 'acme.io', 'widgets'],
          ['kubernetes', '$context', 'acme.io', '$widget'],
        ],
      },
    },
    {
      name: 'a specs.json resource without an entry in links keeps its list and detail nodes, also when its group version has an empty links entry',
      input:: function()
        local spec = {
          paths: {
            '/apis/acme.io/v1/widgets': { get: { 'x-kubernetes-group-version-kind': { kind: 'Widget' } } },
            '/apis/acme.io/v1/widgets/{name}': { get: {} },
          },
        };
        local entries = resourceEntries(arcourseKubernetes.graph {
          manifest: false,
          contexts: [],
          specs: { 'apis/acme.io/v1': spec },
          links: { 'acme.io/v1': [] },
        });
        { paths: [nodePath(e) for e in entries[2:]] },
      expected: {
        paths: [
          ['kubernetes', '$context', 'acme.io', 'widgets'],
          ['kubernetes', '$context', 'acme.io', '$widget'],
        ],
      },
    },
    {
      name: 'a group resource detail node gets an extra linkSpec from a matching inferred link, rewritten from the raw OpenAPI target path into the node-route convention',
      input:: function()
        local daemonset = { name: 'daemonsets', kind: 'DaemonSet', namespaced: true, verbs: ['list', 'get'], version: 'v1' };
        local controllerrevisions = { name: 'controllerrevisions', kind: 'ControllerRevision', namespaced: true, verbs: ['list', 'get'], version: 'v1' };
        local links = [
          {
            at: ['metadata', 'ownerReferences'],
            keys: [{ const: 'owner' }, { path: ['name'] }],
            sourcePath: '/apis/apps/v1/namespaces/{namespace}/controllerrevisions/{name}',
            value: [
              { const: 'apis' }, { const: 'apps' }, { const: 'v1' }, { const: 'namespaces' }, { origin: 'namespace' },
              { const: 'daemonsets' }, { param: 'name', path: ['name'] },
            ],
          },
        ];
        local entries = nodes([{ group: 'apps', resources: [daemonset, controllerrevisions], links: links }]);
        local detail = [e for e in entries if nodePath(e) == ['kubernetes', '$context', '$namespace', 'apps', '$controllerrevision']][0];
        local body = nodeBody(detail);
        {
          fieldNames: fieldNames(body),
          linkSpecs: manifestLiteral(field(body, 'linkSpecs')),
        },
      expected: {
        fieldNames: ['response', 'data', 'linkSpecs'],
        linkSpecs: [
          {
            at: ['metadata', 'ownerReferences'],
            keys: [{ const: 'owner' }, { path: ['name'] }],
            value: [
              { const: 'kubernetes' },
              { origin: 'context' },
              { origin: 'namespace' },
              { const: 'apps' },
              { param: 'daemonset', path: ['name'] },
            ],
          },
        ],
      },
    },
    {
      name: 'a core (groupless) resource detail node gets a linkSpec whose value has no group segment and drops the leading "api"/version pair instead of "apis"/group/version',
      input:: function()
        local pod = { name: 'pods', kind: 'Pod', namespaced: true, verbs: ['get'], version: 'v1' };
        local secret = { name: 'secrets', kind: 'Secret', namespaced: true, verbs: ['list', 'get'], version: 'v1' };
        local links = [
          {
            at: ['spec', 'volumes'],
            keys: [{ path: ['name'] }],
            sourcePath: '/api/v1/namespaces/{namespace}/pods/{name}',
            value: [
              { const: 'api' }, { const: 'v1' }, { const: 'namespaces' }, { origin: 'namespace' },
              { const: 'secrets' }, { param: 'name', path: ['secretName'] },
            ],
          },
        ];
        local entries = nodes([{ group: '', resources: [pod, secret], links: links }]);
        local detail = [e for e in entries if nodePath(e) == ['kubernetes', '$context', '$namespace', '$pod']][0];
        manifestLiteral(field(nodeBody(detail), 'linkSpecs'))[0].value,
      expected: [
        { const: 'kubernetes' },
        { origin: 'context' },
        { origin: 'namespace' },
        { param: 'secret', path: ['secretName'] },
      ],
    },
    {
      name: 'list nodes never get inferred linkSpecs (only their fixed self-link), and a detail-node link whose sourcePath does not match is not attached',
      input:: function()
        local resource = { name: 'widgets', kind: 'Widget', namespaced: false, verbs: ['list', 'get'], version: 'v1' };
        local links = [
          { at: ['items'], keys: [], sourcePath: '/apis/acme.io/v1/widgets', value: [] },
          { at: [], keys: [], sourcePath: '/apis/acme.io/v1/unrelated/{name}', value: [] },
        ];
        local entries = nodes([{ group: 'acme.io', resources: [resource], links: links }]);
        local list = [e for e in entries if nodePath(e) == ['kubernetes', '$context', 'acme.io', 'widgets']][0];
        local detail = [e for e in entries if nodePath(e) == ['kubernetes', '$context', 'acme.io', '$widget']][0];
        {
          listLinkSpecCount: std.length(manifestLiteral(field(nodeBody(list), 'linkSpecs'))),
          detailFieldNames: fieldNames(nodeBody(detail)),
        },
      expected: {
        listLinkSpecCount: 1,
        detailFieldNames: ['response', 'data'],
      },
    },
    {
      name: 'a link pointing directly at a Namespace object gets its param renamed to "namespace" (a well-known cross-group target), while a link to an unresolvable cross-group resource is dropped rather than emitting a broken param',
      input:: function()
        local resourcequota = { name: 'resourcequotas', kind: 'ResourceQuota', namespaced: true, verbs: ['list', 'get'], version: 'v1' };
        local links = [
          {
            at: [],
            keys: [{ path: ['metadata', 'name'] }],
            sourcePath: '/api/v1/namespaces/{namespace}/resourcequotas/{name}',
            value: [
              { const: 'api' }, { const: 'v1' }, { const: 'namespaces' }, { param: 'name', path: ['metadata', 'namespace'] },
            ],
          },
          {
            at: [],
            keys: [{ path: ['metadata', 'name'] }],
            sourcePath: '/api/v1/namespaces/{namespace}/resourcequotas/{name}',
            value: [
              { const: 'apis' }, { const: 'acme.io' }, { const: 'v1' }, { const: 'namespaces' }, { origin: 'namespace' },
              { const: 'widgets' }, { param: 'name', path: ['spec', 'widgetName'] },
            ],
          },
        ];
        local entries = nodes([{ group: '', resources: [resourcequota], links: links }]);
        local detail = [e for e in entries if nodePath(e) == ['kubernetes', '$context', '$namespace', '$resourcequota']][0];
        manifestLiteral(field(nodeBody(detail), 'linkSpecs')),
      expected: [
        {
          at: [],
          keys: [{ path: ['metadata', 'name'] }],
          value: [
            { const: 'kubernetes' },
            { origin: 'context' },
            { param: 'namespace', path: ['metadata', 'namespace'] },
          ],
        },
      ],
    },
    {
      name: 'a namespace linkSpec (isolated from the default ownerReference one) is applied to both namespaced and cluster-scoped detail nodes ahead of any inferred linkSpecs',
      input:: function()
        local namespacedResource = { name: 'widgets', kind: 'Widget', namespaced: true, verbs: ['get'], version: 'v1' };
        local clusterResource = { name: 'gadgets', kind: 'Gadget', namespaced: false, verbs: ['get'], version: 'v1' };
        local namespaceOnlyLinkSpec = {
          at: ['metadata'],
          keys: [{ const: 'namespace' }],
          value: [{ const: 'kubernetes' }, { origin: 'context' }, { param: 'namespace', path: ['namespace'] }],
        };
        local entries = unwrap(arcourseKubernetes.graph {
          manifest: false,
          globalLinkSpecs: [namespaceOnlyLinkSpec],
          data+: { groups: [{ group: '', resources: [namespacedResource, clusterResource] }] },
        }._view.jsonnet).elements;
        local namespacedDetail = [e for e in entries if nodePath(e) == ['kubernetes', '$context', '$namespace', '$widget']][0];
        local clusterDetail = [e for e in entries if nodePath(e) == ['kubernetes', '$context', '$gadget']][0];
        {
          namespacedLinkSpecs: manifestLiteral(field(nodeBody(namespacedDetail), 'linkSpecs')),
          clusterLinkSpecs: manifestLiteral(field(nodeBody(clusterDetail), 'linkSpecs')),
        },
      expected: {
        namespacedLinkSpecs: [{
          at: ['metadata'],
          keys: [{ const: 'namespace' }],
          value: [{ const: 'kubernetes' }, { origin: 'context' }, { param: 'namespace', path: ['namespace'] }],
        }],
        clusterLinkSpecs: [{
          at: ['metadata'],
          keys: [{ const: 'namespace' }],
          value: [{ const: 'kubernetes' }, { origin: 'context' }, { param: 'namespace', path: ['namespace'] }],
        }],
      },
    },
    {
      name: 'graph defaults globalLinkSpecs to a namespace linkSpec plus an ownerReference linkSpec that computes its group/method from the item, source held as parseable jsonnet text rather than a live function',
      input:: function()
        local specs = arcourseKubernetes.graph.globalLinkSpecs;
        {
          count: std.length(specs),
          ownerReference: specs[1],
        },
      expected: {
        count: 2,
        ownerReference: {
          at: ['metadata', 'ownerReferences'],
          keys: [{ const: 'owner' }, { path: ['name'] }],
          value: [
            { const: 'kubernetes' },
            { origin: 'context' },
            { origin: 'namespace' },
            { path: ['apiVersion'], transform: "function(v) local p = std.split(v, '/'); if std.length(p) > 1 then p[0] else ''" },
            { param: { path: ['kind'], transform: 'std.asciiLower' }, path: ['name'] },
          ],
        },
      },
    },
    {
      name: 'globalLinkSpecs can be overridden by the caller to add more or to clear the default',
      input:: function()
        local resource = { name: 'widgets', kind: 'Widget', namespaced: true, verbs: ['get'], version: 'v1' };
        local extraLinkSpec = {
          at: [],
          keys: [{ const: 'owner' }],
          value: [{ const: 'kubernetes' }, { origin: 'context' }, { const: 'acme.io' }, { param: 'owner', path: ['metadata', 'annotations', 'acme.io/owner'] }],
        };
        local withExtra = nodes([{ group: '', resources: [resource] }], [{ at: ['metadata'], keys: [{ const: 'namespace' }], value: [{ const: 'kubernetes' }, { origin: 'context' }, { param: 'namespace', path: ['namespace'] }] }, extraLinkSpec]);
        local withNone = nodes([{ group: '', resources: [resource] }], []);
        local detailFrom(entries) = [e for e in entries if nodePath(e) == ['kubernetes', '$context', '$namespace', '$widget']][0];
        {
          extraCount: std.length(manifestLiteral(field(nodeBody(detailFrom(withExtra)), 'linkSpecs'))),
          noneFieldNames: fieldNames(nodeBody(detailFrom(withNone))),
        },
      expected: {
        extraCount: 2,
        noneFieldNames: ['response', 'data'],
      },
    },
  ],
}
