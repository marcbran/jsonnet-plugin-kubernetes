{
  podTableNode:: {
    table+:: {
      columns: [
        { label: 'Name', path: ['metadata', 'name'] },
        { label: 'Namespace', path: ['metadata', 'namespace'] },
        {
          label: 'Ready',
          paths: [['spec', 'containers'], ['status', 'containerStatuses']],
          value(item)::
            local statuses = std.get(item.status, 'containerStatuses', []);
            '%d/%d' % [std.length([s for s in statuses if s.ready]), std.length(item.spec.containers)],
        },
        { label: 'Status', path: ['status', 'phase'] },
        {
          label: 'Restarts',
          paths: [['status', 'containerStatuses']],
          value(item)::
            std.foldl(function(acc, s) acc + s.restartCount, std.get(item.status, 'containerStatuses', []), 0),
        },
        { label: 'Node', path: ['spec', 'nodeName'] },
        { label: 'Pod IP', path: ['status', 'podIP'] },
        { label: 'Created', path: ['metadata', 'creationTimestamp'] },
      ],
    },
  },
  nodeList: [
    [['kubernetes']],
    [['kubernetes', '$context', 'pods'], $.podTableNode],
    [['kubernetes', '$context', '$namespace', 'pods'], $.podTableNode],
  ],
}
