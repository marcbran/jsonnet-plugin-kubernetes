local j = import 'jsonnet/main.libsonnet';
local kubernetes = import '../../jsonnet-plugin-kubernetes/main.libsonnet';

local fieldListMixin = |||
  {
    kubernetes+: {
      table+: {
        fieldList(columns, linkSpecs)::
          std.set([
            std.join('.', path)
            for col in columns
            for path in std.get(col, 'paths', [col.path])
          ] + [
            std.join('.', seg.path)
            for spec in linkSpecs
            if std.isArray(spec.value)
            for seg in std.get(spec, 'keys', []) + spec.value
            if std.objectHas(seg, 'path')
          ]),
      },
    },
  }
|||;

local resourceVerbs(resource) = std.get(resource, 'verbs', []);

local mergeResource(left, right) =
  if left.kind != right.kind then
    error 'conflicting kind for %s: %s vs %s' % [left.name, left.kind, right.kind]
  else
    left { namespaced: left.namespaced || right.namespaced, verbs: std.set(resourceVerbs(left) + resourceVerbs(right)) };

local dedupeResources(resources) =
  local byKey = std.foldl(
    function(acc, r)
      acc { [r.name]: if std.objectHas(acc, r.name) then mergeResource(acc[r.name], r) else r },
    resources,
    {}
  );
  [byKey[k] for k in std.sort(std.objectFields(byKey))];

local namespaceLinkSpec = {
  at: ['metadata'],
  keys: [{ const: 'namespace' }],
  value: [{ const: 'kubernetes' }, { origin: 'context' }, { param: 'namespace', path: ['namespace'] }],
};

local ownerReferenceLinkSpec = {
  at: ['metadata', 'ownerReferences'],
  keys: [{ const: 'owner' }, { path: ['name'] }],
  value: [
    { const: 'kubernetes' },
    { origin: 'context' },
    { origin: 'namespace' },
    { path: ['apiVersion'], transform: "function(v) local p = std.split(v, '/'); if std.length(p) > 1 then p[0] else ''" },
    { param: { path: ['kind'], transform: 'std.asciiLower' }, path: ['name'] },
  ],
};

local isSubdomainPattern(pattern) = std.startsWith(pattern, '*.');

local matchesGroup(pattern, group) =
  if pattern == '*' then true
  else if isSubdomainPattern(pattern) then std.endsWith(group, pattern[1:])
  else group == pattern || std.endsWith(group, '.' + pattern);

local patternSpecificity(pattern) =
  if pattern == '*' then 0
  else if isSubdomainPattern(pattern) then std.length(std.split(pattern[2:], '.')) + 0.5
  else std.length(std.split(pattern, '.'));

local bestMatch(patterns, group) =
  local matches = [
    { index: i, specificity: patternSpecificity(patterns[i]) }
    for i in std.range(0, std.length(patterns) - 1)
    if matchesGroup(patterns[i], group)
  ];
  if std.length(matches) == 0 then null
  else std.sort(matches, function(m) [-m.specificity, m.index])[0];

local groupLayout(groups, favorites, hidden) =
  local category(group) = if std.length(std.findSubstr('.', group)) == 0 then 0 else 1;
  local favoriteRank(group) =
    local m = bestMatch(favorites, group);
    if m == null then std.length(favorites) else m.index;
  local isHidden(group) =
    local h = bestMatch(hidden, group);
    local f = bestMatch(favorites, group);
    h != null && (f == null || h.specificity > f.specificity);
  local ordered = std.sort(groups, function(group) [favoriteRank(group), category(group)] + std.reverse(std.split(group, '.')));
  {
    expanded: [g for g in ordered if !isHidden(g)],
    hidden: [g for g in ordered if isHidden(g)],
  };

local listObject(settings, groupNames, indent) =
  local le(n) = j.Fodder.LineEnd(0, n);
  local layout = groupLayout(groupNames, settings.favoriteGroups, settings.hiddenGroups);
  local stringArray(values, i) = j.Array([j.String(v).fodder(le(i + 2)) for v in values]).closeFodder(le(i));
  local hiddenItems = settings.hiddenResources;
  local fields =
    [j.Field('columns', j.Number(std.toString(settings.listColumns)))] +
    (if std.length(layout.expanded) > 0 then [j.Field('groupOrder', stringArray(layout.expanded, indent + 2))] else []) +
    (if std.length(layout.hidden) > 0 then [j.Field('hiddenGroups', stringArray(layout.hidden, indent + 2))] else []) +
    (if std.length(hiddenItems) > 0 then [j.Field('hiddenItems', j.Object([
       j.Field(j.String(title), stringArray(hiddenItems[title], indent + 4)) { fodder: [le(indent + 4)] }
       for title in std.objectFields(hiddenItems)
     ]).closeFodder(le(indent + 2)))] else []);
  if std.length(fields) == 1 then j.Object(fields)
  else j.Object([field { fodder: [le(indent + 2)] } for field in fields]).closeFodder(le(indent));

local namespaceListObject(settings, namespaces, groupNames) =
  local le(n) = j.Fodder.LineEnd(0, n);
  local defaultObject = listObject(settings, groupNames, 4);
  if std.length(namespaces) == 0 then defaultObject
  else j.Std.get(
    j.Object([
      j.Field(j.String(namespace), listObject(settings + namespaces[namespace], groupNames, 6)) { fodder: [le(6)] }
      for namespace in std.objectFields(namespaces)
    ]).closeFodder(le(4)),
    j.Member(j.Dollar, 'namespace'),
  ).default(defaultObject);

local listField(expr) = j.Field('list', expr) { Hide: 0 };

local isAsciiLetter(c) = (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z');
local isAsciiDigit(c) = c >= '0' && c <= '9';
local isJsonnetIdent(s) =
  if std.length(s) == 0 then false
  else
    local len = std.length(s);
    local identStart(c) = c == '_' || isAsciiLetter(c);
    local identPart(c) = identStart(c) || isAsciiDigit(c);
    local check(i) =
      if i >= len then true
      else if i == 0 then identStart(s[i]) && check(i + 1)
      else identPart(s[i]) && check(i + 1);
    check(0);
local jsonnetKeywords = [
  'assert', 'else', 'error', 'false', 'for', 'function', 'if',
  'import', 'importstr', 'in', 'local', 'null', 'self', 'super',
  'tailstrict', 'then', 'true',
];
local isJsonnetKeyword(s) = std.member(jsonnetKeywords, s);
local isUnquotedFieldName(s) = isJsonnetIdent(s) && !isJsonnetKeyword(s);
local access(expr, name) =
  if isUnquotedFieldName(name) then j.Member(expr, name) else j.Index(expr, j.String(name));

local objectField(name, expr) =
  if isUnquotedFieldName(name) then j.Field(name, expr) else j.Field(j.String(name), expr);

local generate(resources, group, links=[], globalLinkSpecs=[], namespaceListField=null) =
  local le(indent=0) = j.Fodder.LineEnd(0, indent);
  local prettyArray(elements, indent=0) =
    j.Array([elem.fodder(le(indent + 2)) for elem in elements]).closeFodder(le(indent));
  local prettyObject(fields, indent=0) =
    j.Object([field { fodder: [le(indent + 2)] } for field in fields]).closeFodder(le(indent));

  local contains(xs, x) = std.length([y for y in xs if y == x]) > 0;
  local hasVerb(resource, verb) = contains(resourceVerbs(resource), verb);
  local lowerKind(resource) = std.asciiLower(resource.kind);
  local hasDuplicateKind(resource) =
    std.length([r for r in resources if r.kind == resource.kind]) > 1;
  local resourcePrefix = if group == '' then [] else [group];
  local itemVar(resource) =
    local base =
      if hasDuplicateKind(resource) && group != '' then
        '%s%s' % [lowerKind(resource), std.substr(std.md5(group), 0, 8)]
      else
        lowerKind(resource);
    local candidate = if isUnquotedFieldName(base) then base else 'item';
    if isJsonnetKeyword(candidate) then candidate + 'Item' else candidate;
  local route(resource) =
    local candidate = resource.name;
    if candidate == itemVar(resource) then candidate + 'List' else candidate;

  local var(name) = j.Var(name);
  local member(expr, name) = j.Member(expr, name);
  local call(expr, args=[]) = j.Apply(expr, args);

  local contextOrigin = { origin: 'context' };
  local groupSegment = if group == '' then [] else [{ const: group }];
  local nameSegment(resource) = { param: itemVar(resource), path: ['metadata', 'name'] };
  local allNamespacesLinkSpec(resource) = {
    at: ['items'],
    keys: [{ path: ['metadata', 'namespace'] }, { path: ['metadata', 'name'] }],
    value: [{ const: 'kubernetes' }, contextOrigin, { param: 'namespace', path: ['metadata', 'namespace'] }]
           + groupSegment + [nameSegment(resource)],
  };
  local singleNamespaceLinkSpec(resource) = {
    at: ['items'],
    keys: [{ path: ['metadata', 'name'] }],
    value: [{ const: 'kubernetes' }, contextOrigin, { origin: 'namespace' }] + groupSegment + [nameSegment(resource)],
  };
  local clusterLinkSpec(resource) = {
    at: ['items'],
    keys: [{ path: ['metadata', 'name'] }],
    value: [{ const: 'kubernetes' }, contextOrigin] + groupSegment + [nameSegment(resource)],
  };

  local apiPrefix(resource) =
    if group == '' then '/api/' + resource.version
    else '/apis/' + group + '/' + resource.version;

  local k8sNeatGet(pathExpr, fieldsExpr=null) =
    call(
      member(member(var('kubernetes'), 'neat'), 'get'),
      [access(j.Dollar, 'context'), pathExpr] + (if fieldsExpr == null then [] else [fieldsExpr])
    );

  local k8sGet(pathExpr, fieldsExpr) =
    call(
      member(var('kubernetes'), 'get'),
      [access(j.Dollar, 'context'), pathExpr, fieldsExpr]
    );

  local staticPath(path) = j.String(path);
  local formatPath(fmt, args) = j.Std.format(j.String(fmt), j.Array(args));
  local toStr(expr) = j.Std.toString(expr);

  local clusterListPath(resource) = staticPath(apiPrefix(resource) + '/' + resource.name);
  local clusterDetailPath(resource) =
    formatPath(apiPrefix(resource) + '/' + resource.name + '/%s', [toStr(access(j.Dollar, itemVar(resource)))]);
  local namespacedAllPath(resource) = staticPath(apiPrefix(resource) + '/' + resource.name);
  local namespacedListPath(resource) =
    formatPath(apiPrefix(resource) + '/namespaces/%s/' + resource.name, [toStr(access(j.Dollar, 'namespace'))]);
  local namespacedDetailPath(resource) =
    formatPath(apiPrefix(resource) + '/namespaces/%s/' + resource.name + '/%s', [
      toStr(access(j.Dollar, 'namespace')),
      toStr(access(j.Dollar, itemVar(resource))),
    ]);

  local clusterDetailPathStr(resource) = apiPrefix(resource) + '/' + resource.name + '/{name}';
  local namespacedDetailPathStr(resource) = apiPrefix(resource) + '/namespaces/{namespace}/' + resource.name + '/{name}';

  local isNamespacesConst(seg) = std.objectHas(seg, 'const') && seg.const == 'namespaces';
  local resourceItemVarByName = { [r.name]: itemVar(r) for r in resources };
  local wellKnownItemVars = { namespaces: 'namespace', nodes: 'node' };
  local targetItemVarFor(name) =
    if std.objectHas(resourceItemVarByName, name) then resourceItemVarByName[name]
    else if std.objectHas(wellKnownItemVars, name) then wellKnownItemVars[name]
    else null;
  local renameParam(seg, name) = if std.objectHas(seg, 'param') then seg { param: name } else seg;
  local rewriteLinkValue(value) =
    local isCore = value[0].const == 'api';
    local targetGroupSegment = if isCore then [] else [value[1]];
    local afterPrefix = std.slice(value, if isCore then 2 else 3, std.length(value), 1);
    local hasNamespace = std.length(afterPrefix) >= 2 && isNamespacesConst(afterPrefix[0]);
    local namespaceValueSeg = if hasNamespace then renameParam(afterPrefix[1], 'namespace') else null;
    local afterNamespace = if hasNamespace then std.slice(afterPrefix, 2, std.length(afterPrefix), 1) else afterPrefix;
    if std.length(afterNamespace) == 0 then
      if namespaceValueSeg == null then null
      else [{ const: 'kubernetes' }, contextOrigin] + targetGroupSegment + [namespaceValueSeg]
    else
      local rawIdSeg = afterNamespace[std.length(afterNamespace) - 1];
      local rawResourceNameSeg = if std.length(afterNamespace) >= 2 then afterNamespace[std.length(afterNamespace) - 2] else null;
      local targetItemVar =
        if rawResourceNameSeg != null && std.objectHas(rawResourceNameSeg, 'const')
        then targetItemVarFor(rawResourceNameSeg.const)
        else null;
      if targetItemVar == null then null
      else
        [{ const: 'kubernetes' }, contextOrigin] +
        (if hasNamespace then [namespaceValueSeg] else []) +
        targetGroupSegment +
        [renameParam(rawIdSeg, targetItemVar)];
  local linksAt(path) =
    local candidates = [l { value: rewriteLinkValue(l.value) } for l in links if l.sourcePath == path];
    [l for l in candidates if l.value != null];

  local compactLiteral(value) =
    if value == null then j.Null
    else if std.type(value) == 'string' then j.String(value)
    else if std.type(value) == 'boolean' then if value then j.True else j.False
    else if std.type(value) == 'number' then j.Number(std.toString(value))
    else if std.type(value) == 'array' then j.Array([compactLiteral(item) for item in value])
    else if std.type(value) == 'object' then j.Object([
      objectField(field, if field == 'transform' then j.parseJsonnet(value[field]) else compactLiteral(value[field]))
      for field in std.objectFields(value)
    ])
    else error 'unsupported literal type: ' + std.type(value);

  local responseFields(expr, hidden=false) = [
    j.Field('response', expr) { Hide: 0 },
    j.Field('data', member(j.Self, 'response')) { Hide: if hidden then 0 else 1 },
  ];
  local linkSpecEntry(spec) =
    prettyObject([objectField(field, compactLiteral(spec[field])) for field in ['at', 'keys', 'value']], 6);
  local dataObject(expr, linkSpecs=[], viewFields=[]) = prettyObject(
    responseFields(expr) +
    (if std.length(linkSpecs) > 0 then [j.Field('linkSpecs', prettyArray([linkSpecEntry(s) for s in linkSpecs], 4)) { Hide: 0 }] else []) +
    viewFields,
    2
  );
  local listObject(dataExpr, linkSpecs, columnsExpr=null) = prettyObject(
    responseFields(dataExpr, true) + [j.Field('linkSpecs', prettyArray([linkSpecEntry(s) for s in linkSpecs], 4)) { Hide: 0 }] +
    (if columnsExpr != null then [j.Field('table', prettyObject([
       j.Field('at', j.Array([j.String('items')])),
       j.Field('columns', columnsExpr),
     ], 4)) { Hide: 0 }] else []),
    2
  );
  local nodeView(name) = member(member(var('a'), name), 'node');
  local nodePath(path) = j.Array([j.String(p) for p in path]);
  local node(path, viewExpr, body) = j.Array([nodePath(path), j.Add(viewExpr, body)]);

  local columnLiteral(col) = compactLiteral({ label: col.label, path: col.path });
  local resourceColumns(resource) =
    local cols = std.get(resource, 'columns', []);
    if std.length(cols) > 0 then
      prettyArray([columnLiteral(col) for col in cols], 4)
    else null;
  local listView(resource) =
    if std.length(std.get(resource, 'columns', [])) > 0 then nodeView('table') else nodeView('list');

  local tableFields = call(
    member(member(member(var('a'), 'kubernetes'), 'table'), 'fieldList'),
    [member(member(j.Dollar, 'table'), 'columns'), member(j.Dollar, 'linkSpecs')]
  );
  local listData(resource, pathExpr) =
    if resourceColumns(resource) != null then k8sGet(pathExpr, tableFields) else k8sNeatGet(pathExpr);

  local namespacedAllList(resource) =
    local linkSpecs = [allNamespacesLinkSpec(resource)];
    local columnsExpr = resourceColumns(resource);
    local data = listData(resource, namespacedAllPath(resource));
    node(
      ['kubernetes', '$context'] + resourcePrefix + [route(resource)],
      listView(resource),
      listObject(data, linkSpecs, columnsExpr)
    );

  local namespacedList(resource) =
    local linkSpecs = [singleNamespaceLinkSpec(resource)];
    local columnsExpr = resourceColumns(resource);
    local data = listData(resource, namespacedListPath(resource));
    node(
      ['kubernetes', '$context', '$namespace'] + resourcePrefix + [route(resource)],
      listView(resource),
      listObject(data, linkSpecs, columnsExpr)
    );

  local namespacedDetail(resource) =
    node(
      ['kubernetes', '$context', '$namespace'] + resourcePrefix + ['$' + itemVar(resource)],
      nodeView('resource'),
      dataObject(k8sNeatGet(namespacedDetailPath(resource)), globalLinkSpecs + linksAt(namespacedDetailPathStr(resource)))
    );

  local clusterList(resource) =
    local linkSpecs = [clusterLinkSpec(resource)];
    local columnsExpr = resourceColumns(resource);
    local data = listData(resource, clusterListPath(resource));
    node(
      ['kubernetes', '$context'] + resourcePrefix + [route(resource)],
      listView(resource),
      listObject(data, linkSpecs, columnsExpr)
    );

  local namespaceViewFields(resource) =
    if group == '' && resource.name == 'namespaces' then [
      j.Field('resource', compactLiteral({ stacked: true })) { Hide: 0 },
      namespaceListField,
    ] else [];
  local clusterDetail(resource) =
    node(
      ['kubernetes', '$context'] + resourcePrefix + ['$' + itemVar(resource)],
      nodeView('resource'),
      dataObject(k8sNeatGet(clusterDetailPath(resource)), globalLinkSpecs + linksAt(clusterDetailPathStr(resource)), namespaceViewFields(resource))
    );

  local resourceNodes(resource) =
    if resource.namespaced then
      (if hasVerb(resource, 'list') then [namespacedAllList(resource), namespacedList(resource)] else []) +
      (if hasVerb(resource, 'get') then [namespacedDetail(resource)] else [])
    else
      (if hasVerb(resource, 'list') then [clusterList(resource)] else []) +
      (if hasVerb(resource, 'get') then [clusterDetail(resource)] else []);

  {
    nodes: std.flattenArrays([resourceNodes(r) for r in resources]),
    contextRoutes: [route(r) for r in resources if hasVerb(r, 'list')],
    namespaceRoutes: [route(r) for r in resources if r.namespaced && hasVerb(r, 'list')],
  };

local generateAll(groups, manifest, globalLinkSpecs, settings, namespaces) =
  local groupNames = std.set([g.group for g in groups if g.group != '']);
  local le(indent=0) = j.Fodder.LineEnd(0, indent);
  local prettyArray(elements, indent=0) =
    j.Array([elem.fodder(le(indent + 2)) for elem in elements]).closeFodder(le(indent));
  local prettyObject(fields, indent=0) =
    j.Object([field { fodder: [le(indent + 2)] } for field in fields]).closeFodder(le(indent));
  local prettyObjectComp(fields, specs, indent=0) =
    j.ObjectComp(
      [field { fodder: [le(indent + 2)] } for field in fields],
      [spec.forFodder(le(indent + 2)) for spec in specs]
    ).closeFodder(le(indent));
  local var(name) = j.Var(name);
  local member(expr, name) = j.Member(expr, name);
  local call(expr, args=[]) = j.Apply(expr, args);

  local contextsNode = j.Array([
    j.Array([j.String('kubernetes'), j.String('contexts')]),
    j.Add(
      member(member(var('a'), 'list'), 'node'),
      prettyObject([
        j.Field('response', call(member(var('kubernetes'), 'contexts'))) { Hide: 0 },
        j.Field('data', member(j.Self, 'response')) { Hide: 0 },
        j.Field('links', prettyObjectComp(
          [j.Field(member(var('c'), 'name'), call(member(member(var('root'), 'kubernetes'), 'context'), [member(var('c'), 'name')]))],
          [j.ForSpec('c', member(j.Dollar, 'data'))],
          4
        )),
      ], 2)
    ),
  ]);
  local contextNode = j.Array([
    j.Array([j.String('kubernetes'), j.String('$context')]),
    prettyObject([listField(listObject(settings, groupNames, 4))], 2),
  ]);

  local results = [
    { group: g.group, result: generate(g.resources, g.group, std.get(g, 'links', []), globalLinkSpecs, listField(namespaceListObject(settings, namespaces, groupNames))) }
    for g in groups
  ];
  local contextScope = call(member(member(var('root'), 'kubernetes'), 'context'), [member(j.Dollar, 'context')]);
  local apiResourcesNode(path, scopeExpr, routesField) =
    local scope = var('scope');
    local routesOf(group) = std.flattenArrays([r.result[routesField] for r in results if r.group == group]);
    local groupLinks(group) = prettyObject([objectField(route, access(access(scope, group), route)) for route in routesOf(group)], 6);
    j.Array([
      j.Array([j.String(p) for p in path]),
      j.Add(
        member(member(var('a'), 'list'), 'node'),
        prettyObject([
          j.FieldLocal('scope', scopeExpr),
          listField(listObject(settings { hiddenGroups: [], hiddenResources: {} }, groupNames, 4)),
          j.Field('links', prettyObject(
            [objectField(route, access(scope, route)) for route in routesOf('')] +
            [objectField(group, groupLinks(group)) for group in groupNames if std.length(routesOf(group)) > 0],
            4
          )),
        ], 2)
      ),
    ]);
  local apiResourcesNodes = [
    apiResourcesNode(['kubernetes', '$context', 'apiResources'], contextScope, 'contextRoutes'),
    apiResourcesNode(
      ['kubernetes', '$context', '$namespace', 'apiResources'],
      call(member(contextScope, 'namespace'), [member(j.Dollar, 'namespace')]),
      'namespaceRoutes'
    ),
  ];

  local allRouteNodes = [contextsNode, contextNode] + std.flattenArrays([r.result.nodes for r in results]) + apiResourcesNodes;
  local binds = [
    j.LocalBind('base', j.Import('arcourse-ui/main.libsonnet')),
    j.LocalBind('a', j.Add(var('base'), j.parseJsonnet(fieldListMixin))),
    j.LocalBind('kubernetes', j.Import('kubernetes/main.libsonnet')),
    j.LocalBind('root', j.Import('root')),
  ];
  local generated = std.foldr(
    function(i, acc)
      local node = j.Local(binds[i], acc);
      if i == 0 then node else node.fodder(j.Fodder.LineEnd(0, 0)),
    std.range(0, std.length(binds) - 1),
    prettyArray(allRouteNodes).fodder(j.Fodder.LineEnd(0, 0))
  );
  if manifest then j.manifestJsonnet(generated) else generated;

local groupVersionFromKey(key) =
  local parts = [p for p in std.split(key, '/') if p != ''];
  if parts[0] == 'api' then { group: '', version: parts[1] }
  else { group: parts[1], version: parts[2] };

local linksFromSpec(spec) =
  local suffix = '/{name}';
  local suffixLen = std.length(suffix);
  local endsWith(s) =
    std.length(s) >= suffixLen &&
    std.substr(s, std.length(s) - suffixLen, suffixLen) == suffix;
  [
    { sourcePath: std.substr(p, 0, std.length(p) - suffixLen), targetPath: p, array: ['items'], vars: { name: ['metadata', 'name'] } }
    for p in std.objectFields(spec.paths)
    if endsWith(p) && std.objectHas(spec.paths, std.substr(p, 0, std.length(p) - suffixLen))
  ];

local resourceNameFromPath(path) =
  local parts = [p for p in std.split(path, '/') if p != ''];
  local last = parts[std.length(parts) - 1];
  local isParam(s) = std.length(s) >= 1 && std.substr(s, 0, 1) == '{';
  if isParam(last) then parts[std.length(parts) - 2] else last;

local isSubresourcePath(path) =
  local parts = [p for p in std.split(path, '/') if p != ''];
  std.length(parts) >= 2 &&
  parts[std.length(parts) - 2] == '{name}' &&
  std.substr(parts[std.length(parts) - 1], 0, 1) != '{';

local resourcesFromSpec(spec, group, version, columns, links) =
  local groupVersion = if group == '' then version else group + '/' + version;
  local columnsForGroup = std.get(columns, groupVersion, []);
  local explicitLinks = std.get(links, groupVersion, []);
  local isNamespacedPath(path) =
    std.length(std.findSubstr('/namespaces/{', path)) > 0;
  local apiPrefix = if group == '' then '/api/' + version else '/apis/' + group + '/' + version;
  local collectionPath(name, namespaced) =
    apiPrefix + (if namespaced then '/namespaces/{namespace}/' else '/') + name;
  local detailPath(name, namespaced) = collectionPath(name, namespaced) + '/{name}';
  local findKind(path) =
    local pathEntry = std.get(spec.paths, path, {});
    local getOp = std.get(pathEntry, 'get', null);
    if getOp == null then null
    else std.get(getOp, 'x-kubernetes-group-version-kind', null);
  local hasGet(path) =
    std.objectHas(spec.paths, path) && std.objectHas(spec.paths[path], 'get');
  local defaultColumns(namespaced) =
    [{ key: 'metadata.name', kind: 'name', label: 'Name', path: ['metadata', 'name'], priority: 'primary' }] +
    (if namespaced then [{ key: 'metadata.namespace', kind: 'text', label: 'Namespace', path: ['metadata', 'namespace'], priority: 'secondary' }] else []) +
    [{ key: 'metadata.creationTimestamp', kind: 'timestamp', label: 'Created', path: ['metadata', 'creationTimestamp'], priority: 'tertiary' }];
  local findColumns(path, namespaced) =
    local matching = [c for c in columnsForGroup if c.sourcePath == path];
    if std.length(matching) > 0 then matching[0].columns else defaultColumns(namespaced);
  local resourceFromLink(link) =
    local name = resourceNameFromPath(link.sourcePath);
    local namespaced = isNamespacedPath(link.sourcePath);
    local gvk = findKind(link.sourcePath);
    {
      name: name,
      kind: if gvk != null then gvk.kind else 'Unknown',
      namespaced: namespaced,
      verbs: (if hasGet(collectionPath(name, namespaced)) then ['list'] else []) + (if hasGet(detailPath(name, namespaced)) then ['get'] else []),
      group: group,
      version: version,
      columns: findColumns(collectionPath(name, namespaced), namespaced),
    };
  {
    resources: dedupeResources([
      resourceFromLink(link)
      for link in linksFromSpec(spec) + explicitLinks
      if !isSubresourcePath(link.sourcePath)
    ]),
    links: explicitLinks,
  };

local defaultColumns(namespaced) =
  [{ key: 'metadata.name', kind: 'name', label: 'Name', path: ['metadata', 'name'], priority: 'primary' }] +
  (if namespaced then [{ key: 'metadata.namespace', kind: 'text', label: 'Namespace', path: ['metadata', 'namespace'], priority: 'secondary' }] else []) +
  [{ key: 'metadata.creationTimestamp', kind: 'timestamp', label: 'Created', path: ['metadata', 'creationTimestamp'], priority: 'tertiary' }];

local resourcesFromDiscovery(discovery, group, columns, links) =
  local groupVersion = if group == '' then 'v1' else discovery.groupVersion;
  local bareVersion =
    local parts = std.split(discovery.groupVersion, '/');
    parts[std.length(parts) - 1];
  local columnsForGroup = std.get(columns, groupVersion, []);
  local linksForGroup = std.get(links, groupVersion, null);
  local apiPrefix = if group == '' then '/api/' + discovery.groupVersion else '/apis/' + discovery.groupVersion;
  local isNamespaced(r) = r.namespaced;
  local sourcePath(r) = if isNamespaced(r) then apiPrefix + '/namespaces/{namespace}/' + r.name else apiPrefix + '/' + r.name;
  local findColumns(r) =
    local path = sourcePath(r);
    local matching = [c for c in columnsForGroup if c.sourcePath == path];
    if std.length(matching) > 0 then matching[0].columns else defaultColumns(isNamespaced(r));
  local effectiveLinks = if linksForGroup != null then linksForGroup else [];
  local linkedNames = std.set([
    local parts = [p for p in std.split(l.sourcePath, '/') if p != ''];
    parts[std.length(parts) - 1]
    for l in effectiveLinks
  ]);
  {
    resources: dedupeResources([
      r { group: group, version: bareVersion, columns: findColumns(r) }
      for r in discovery.resources
      if std.length(std.findSubstr('/', r.name)) == 0
    ]),
    links: effectiveLinks,
  };

local mergeGroups(groups) =
  local byKey = std.foldl(
    function(acc, g)
      local key = g.group;
      acc {
        [key]: if std.objectHas(acc, key)
          then acc[key] {
            resources: dedupeResources(acc[key].resources + g.resources),
            links: std.get(acc[key], 'links', []) + std.get(g, 'links', []),
          }
          else g,
      },
    groups,
    {}
  );
  [byKey[k] for k in std.objectFields(byKey)];

local groupVersionsOrdered(g) =
  [g.preferredVersion] + [v for v in g.versions if v.version != g.preferredVersion.version];

local groupsFromContext(ctx, columns, links) =
  local core = kubernetes.get(ctx, '/api/v1');
  local apis = kubernetes.get(ctx, '/apis');
  local coreResult = resourcesFromDiscovery(core, '', columns, links);
  [{ group: '', resources: coreResult.resources, links: coreResult.links }] + std.flattenArrays([
    [
      local discovery = kubernetes.get(ctx, '/apis/' + v.groupVersion);
      local result = resourcesFromDiscovery(discovery, g.name, columns, links);
      { group: g.name, resources: result.resources, links: result.links }
      for v in groupVersionsOrdered(g)
    ]
    for g in apis.groups
  ]);

local groupsFromSpecs(specs, columns, links) =
  mergeGroups([
    local gv = groupVersionFromKey(key);
    local result = resourcesFromSpec(specs[key], gv.group, gv.version, columns, links);
    { group: gv.group, resources: result.resources, links: result.links }
    for key in std.objectFields(specs)
  ]);

{
  _record: false,
  manifest:: true,
  favoriteGroups:: [],
  hiddenGroups:: [],
  hiddenResources:: {},
  listColumns:: 4,
  namespaces:: {},
  contexts:: [],
  specs:: {},
  globalLinkSpecs:: [namespaceLinkSpec, ownerReferenceLinkSpec],
  data::
    local columns = if std.objectHasAll(self, 'columns') then self.columns else {};
    local links = if std.objectHasAll(self, 'links') then self.links else {};
    {
      groups: mergeGroups(
        std.flattenArrays([groupsFromContext(ctx, columns, links) for ctx in $.contexts]) +
        groupsFromSpecs($.specs, columns, links)
      ),
    },
  _view:: {
    jsonnet: generateAll($.data.groups, $.manifest, $.globalLinkSpecs, {
      listColumns: $.listColumns,
      favoriteGroups: $.favoriteGroups,
      hiddenGroups: $.hiddenGroups,
      hiddenResources: $.hiddenResources,
    }, $.namespaces),
  },
}
