local telemetry = import 'arcourse-telemetry/main.libsonnet';

local workloadStateTimeline = {
  local t = self,
  local labelled(name, expr) = 'label_replace(0 * (%s) + 1, "state", "%s", "", "")' % [expr, name],
  workload:: error 'workloadStateTimeline requires workload',
  workloadTitle:: error 'workloadStateTimeline requires workloadTitle',
  desiredMetric:: error 'workloadStateTimeline requires desiredMetric',
  leafLink:: 'pods',
  availableMetric:: error 'workloadStateTimeline requires availableMetric',
  readyMetric:: error 'workloadStateTimeline requires readyMetric',
  series(metric, extra=''):: 'max by (cluster, namespace, %s) (%s{%s%s})' % [t.workload, metric, t.matchers, extra],
  desired:: t.series(t.desiredMetric),
  conditions:: error 'workloadStateTimeline requires conditions',
  states:: error 'workloadStateTimeline requires states',
  unhealthy:: error 'workloadStateTimeline requires unhealthy',
  expr:: 'max by (%s, state) (%s)' % [t.groupBy, std.join(' or ', [labelled(s, t.conditions[s]) for s in t.states])],
  colorBy:: '{{state}}',
  rowBy:: '{{%s}}' % t.workload,
  labelBy:: '{{state}}',
  tooltip:: '{{%s}} · {{state}} · {{from}} – {{to}}' % t.workload,
  levels:: {
    root: self.context,
    context: {
      expr:: 'count by (%s, state) (%s)' % [self.groupBy, std.join(' or ', [labelled(s, self.conditions[s]) for s in self.unhealthy])],
      rowBy:: '{{%s}}: {{state}}' % self.groupBy,
      labelBy:: '{{value}}',
      tooltip:: '{{%s}} · {{state}}: {{value}} %ss · {{from}} – {{to}}' % [self.groupBy, self.workload],
    },
  },
};

local capacityTimeline = workloadStateTimeline {
  local t = self,
  local available = '(%s or %s)' % [t.series(t.availableMetric), t.series(t.readyMetric)],
  local heldDesired = 'max by (cluster, namespace, %s) (min_over_time(%s{%s}[5m]))' % [t.workload, t.desiredMetric, t.matchers],
  title:: '%s Capacity %s' % [t.workloadTitle, t.titleGroupBy],
  zeroState:: 'ScaledToZero',
  states:: ['Full', 'ScalingUp', 'Degraded', 'Down', t.zeroState],
  unhealthy:: ['Degraded', 'Down'],
  conditions:: {
    Full: '(%s >= %s) and (%s > 0)' % [available, t.desired, t.desired],
    ScalingUp: '(%s >= %s) < %s' % [available, heldDesired, t.desired],
    Degraded: '(%s > 0) < %s' % [available, heldDesired],
    Down: '(%s == 0) and (%s > 0)' % [available, heldDesired],
    [t.zeroState]: '%s == 0' % t.desired,
  },
  colors:: {
    Full: '#5b8a5b',
    ScalingUp: '#5a7a94',
    Degraded: '#b0902f',
    Down: '#a35454',
    [t.zeroState]: '#7a7a7a',
  },
  levels+:: {
    context+: { title:: 'Unhealthy %s Capacity %s' % [self.workloadTitle, self.titleGroupBy] },
  },
};

local deploymentWorkload = {
  workload:: 'deployment',
  workloadTitle:: 'Deployment',
  desiredMetric:: 'kube_deployment_spec_replicas',
  availableMetric:: 'kube_deployment_status_replicas_available',
  readyMetric:: 'kube_deployment_status_replicas_ready',
  podOwners:: 'max by (cluster, namespace, pod) (label_replace(kube_pod_owner{cluster="%(context)s", namespace="%(namespace)s", owner_kind="ReplicaSet"}, "replicaset", "$1", "owner_name", "(.*)") * on (cluster, namespace, replicaset) group_left () max by (cluster, namespace, replicaset) (kube_replicaset_owner{cluster="%(context)s", namespace="%(namespace)s", owner_kind="Deployment", owner_name="%(deployment)s"}))',
};

local statefulsetWorkload = {
  workload:: 'statefulset',
  workloadTitle:: 'StatefulSet',
  desiredMetric:: 'kube_statefulset_replicas',
  availableMetric:: 'kube_statefulset_status_replicas_available',
  readyMetric:: 'kube_statefulset_status_replicas_ready',
  podOwners:: 'max by (cluster, namespace, pod) (kube_pod_owner{cluster="%(context)s", namespace="%(namespace)s", owner_kind="StatefulSet", owner_name="%(statefulset)s"})',
};

local daemonsetWorkload = {
  workload:: 'daemonset',
  workloadTitle:: 'DaemonSet',
  desiredMetric:: 'kube_daemonset_status_desired_number_scheduled',
  availableMetric:: 'kube_daemonset_status_number_available',
  readyMetric:: 'kube_daemonset_status_number_ready',
  zeroState:: 'NoNodes',
  podOwners:: 'max by (cluster, namespace, pod) (kube_pod_owner{cluster="%(context)s", namespace="%(namespace)s", owner_kind="DaemonSet", owner_name="%(daemonset)s"})',
};

local deploymentRolloutTimeline = workloadStateTimeline + deploymentWorkload {
  local t = self,
  local updated = t.series('kube_deployment_status_replicas_updated'),
  local current = t.series('kube_deployment_status_replicas'),
  local paused = '(%s == 1)' % t.series('kube_deployment_spec_paused'),
  local stuck = '(%s == 1)' % t.series('kube_deployment_status_condition', ', condition="Progressing", status="false", reason="ProgressDeadlineExceeded"'),
  title:: 'Deployment Rollout %s' % t.titleGroupBy,
  states:: ['Settled', 'RollingOut', 'Stuck', 'Paused'],
  unhealthy:: ['Stuck'],
  conditions:: {
    Settled: '((%s == %s) and (%s == %s)) unless %s unless %s' % [updated, t.desired, current, t.desired, paused, stuck],
    RollingOut: '((%s < %s) or (%s > %s)) unless %s unless %s' % [updated, t.desired, current, t.desired, paused, stuck],
    Stuck: '%s unless %s' % [stuck, paused],
    Paused: paused,
  },
  colors:: {
    Settled: '#5b8a5b',
    RollingOut: '#5a7a94',
    Stuck: '#a35454',
    Paused: '#7a7a7a',
  },
  levels+:: {
    context+: { title:: 'Stuck Deployment Rollouts %s' % self.titleGroupBy },
  },
};

local statefulsetRolloutTimeline = workloadStateTimeline + statefulsetWorkload {
  local t = self,
  local outdated = 'max by (cluster, namespace, statefulset) (kube_statefulset_status_update_revision{%s} unless on (cluster, namespace, statefulset, revision) kube_statefulset_status_current_revision{%s})' % [t.matchers, t.matchers],
  title:: 'StatefulSet Rollout %s' % t.titleGroupBy,
  hidden:: ['root', 'context'],
  states:: ['Settled', 'Outdated'],
  unhealthy:: [],
  conditions:: {
    Settled: '%s unless on (cluster, namespace, statefulset) %s' % [t.desired, outdated],
    Outdated: outdated,
  },
  colors:: {
    Settled: '#5b8a5b',
    Outdated: '#5a7a94',
  },
};

local daemonsetRolloutTimeline = workloadStateTimeline + daemonsetWorkload {
  local t = self,
  local updated = t.series('kube_daemonset_status_updated_number_scheduled'),
  title:: 'DaemonSet Rollout %s' % t.titleGroupBy,
  hidden:: ['root', 'context'],
  states:: ['Settled', 'Outdated'],
  unhealthy:: [],
  conditions:: {
    Settled: '%s >= %s' % [updated, t.desired],
    Outdated: '%s < %s' % [updated, t.desired],
  },
  colors:: {
    Settled: '#5b8a5b',
    Outdated: '#5a7a94',
  },
};

local podStateTimeline = {
  local t = self,
  local key = 'cluster, namespace, pod',
  local series(metric, extra='') = 'max by (%s) (%s{%s%s})' % [key, metric, t.podMatchers, extra],
  local phase(name) = '(%s == 1)' % series('kube_pod_status_phase', ', phase="%s"' % name),
  local deletedAt = series('kube_pod_deletion_timestamp'),
  local age = '(time() - %s)' % series('kube_pod_created'),
  local notReady = '(%s == 0)' % series('kube_pod_status_ready', ', condition="true"'),
  local unscheduled = '(%s == 1)' % series('kube_pod_status_scheduled', ', condition="false"'),
  local scheduledPending = '(%s unless on (%s) %s)' % [phase('Pending'), key, unscheduled],
  local runningNotReady = '(%s and on (%s) %s)' % [phase('Running'), key, notReady],
  local young = '(%s < 300)' % age,
  local old = '(%s >= 300)' % age,
  local alive(expr) = '((%s) unless on (%s) %s)' % [expr, key, deletedAt],
  local reason = 'topk by (%s) (1, %s or %s or %s)' % [
    key,
    'max by (%s, reason) (kube_pod_status_reason{%s} == 1)' % [key, t.podMatchers],
    'max by (%s, reason) (kube_pod_container_status_waiting_reason{%s} == 1)' % [key, t.podMatchers],
    'max by (%s, reason) (kube_pod_init_container_status_waiting_reason{%s} == 1)' % [key, t.podMatchers],
  ],
  local labelled(name, expr) = 'label_replace(0 * (%s) + 1, "state", "%s", "", "")' % [expr, name],
  local withReason(expr) = '((%s) * on (%s) group_left (reason) %s or ((%s) unless on (%s) %s))' % [expr, key, reason, expr, key, reason],
  local filtered(expr) = if t.podFilter == null then expr else '(%s) and on (%s) %s' % [expr, key, t.podFilter],
  local perPod(states) = 'max by (%s, state, reason) (%s)' % [t.podGroupBy, withReason(filtered(std.join(' or ', [labelled(s, t.conditions[s]) for s in states])))],
  podMatchers:: t.matchers,
  podGroupBy:: t.groupBy,
  podFilter:: null,
  conditions:: {
    StuckTerminating: '(time() - %s) >= 300' % deletedAt,
    Terminating: '(time() - %s) < 300' % deletedAt,
    Succeeded: alive(phase('Succeeded')),
    Failed: alive(phase('Failed')),
    Unknown: alive(phase('Unknown')),
    Unschedulable: alive('(%s and on (%s) %s) and on (%s) %s' % [phase('Pending'), key, unscheduled, key, old]),
    Starting: alive('(%s or %s or (%s and on (%s) %s)) and on (%s) %s' % [scheduledPending, runningNotReady, phase('Pending'), key, unscheduled, key, young]),
    Pending: alive('%s and on (%s) %s' % [scheduledPending, key, old]),
    NotReady: alive('%s and on (%s) %s' % [runningNotReady, key, old]),
    Ready: alive('%s unless on (%s) %s' % [phase('Running'), key, notReady]),
  },
  states:: ['StuckTerminating', 'Terminating', 'Succeeded', 'Failed', 'Unknown', 'Unschedulable', 'Starting', 'Pending', 'NotReady', 'Ready'],
  unhealthy:: ['StuckTerminating', 'Failed', 'Unknown', 'Unschedulable', 'Pending', 'NotReady'],
  title:: 'Pod State %s' % t.titleGroupBy,
  expr:: perPod(t.states),
  colorBy:: '{{state}}',
  rowBy:: '{{pod}}',
  labelBy:: '{{state}}',
  tooltip:: '{{pod}} · {{state}} {{reason}} · {{from}} – {{to}}',
  colors:: {
    Ready: '#5b8a5b',
    Starting: '#5a7a94',
    Succeeded: '#5a7a94',
    Terminating: '#7a7a7a',
    Pending: '#b0902f',
    Unschedulable: '#b0902f',
    NotReady: '#a35454',
    Failed: '#a35454',
    StuckTerminating: '#a35454',
    Unknown: '#7a7a7a',
  },
  levels:: {
    root: self.context,
    context: {
      expr:: 'count by (%s, state) (%s)' % [self.groupBy, std.join(' or ', [labelled(s, self.conditions[s]) for s in self.unhealthy])],
      title:: 'Unhealthy Pods %s' % self.titleGroupBy,
      rowBy:: '{{%s}}: {{state}}' % self.groupBy,
      labelBy:: '{{value}}',
      tooltip:: '{{%s}} · {{state}}: {{value}} pods · {{from}} – {{to}}' % self.groupBy,
    },
    namespace: {
      expr:: perPod([s for s in self.states if s != 'Ready' && s != 'Succeeded']),
      title:: 'Pods Not Ready %s' % self.titleGroupBy,
    },
  },
};

local workloadPodsTimeline = podStateTimeline {
  local t = self,
  title:: '%s Pods' % t.workloadTitle,
  hidden:: ['root', 'context', 'namespace'],
  podMatchers:: 'cluster="%(context)s", namespace="%(namespace)s"',
  podGroupBy:: 'pod',
  podFilter:: t.podOwners,
};

local containerSeries = {
  local t = self,
  key:: 'cluster, namespace, pod, container',
  usage(metric):: 'max by (%s) (%s{%s, %s})' % [t.key, metric, t.matchers, t.usageMatchers],
  usageRate(metric):: 'max by (%s) (rate(%s{%s, %s}[5m]))' % [t.key, metric, t.matchers, t.usageMatchers],
  state(expr):: 'max by (%s) (%s)' % [t.key, expr],
  withBounds(usage, resource):: std.join(' or ', [
    'label_replace(%s, "series", "usage", "", "")' % usage,
    'label_replace(%s, "series", "request", "", "")' % t.state('kube_pod_container_resource_requests{%s, resource="%s"}' % [t.matchers, resource]),
    'label_replace(%s, "series", "limit", "", "")' % t.state('kube_pod_container_resource_limits{%s, resource="%s"}' % [t.matchers, resource]),
  ]),
  worst(expr):: 'max by (%s) (%s)' % [t.groupBy, expr],
  worstTitle:: if t.groupBy == 'container' then '' else 'of Worst Container ',
};

local nodeSeries = {
  local t = self,
  nodes:: 'max by (cluster, node) (kube_node_info{%s})' % t.matchers,
  fromNodeExporter(expr, by=''):: '(max by (cluster, node%s) ((%s) * on (cluster, instance) group_left (node) %s) and on (cluster, node) %s)' % [by, expr, t.nodeExporterInfo, t.nodes],
  activePods:: '(max by (cluster, namespace, pod) (kube_pod_status_phase{phase=~"Pending|Running"}) == 1)',
  allocatable(resource):: 'max by (cluster, node) (kube_node_status_allocatable{%s, resource="%s"})' % [t.matchers, resource],
  requested(resource)::
    local podKey = 'cluster, namespace, pod, node';
    local containers = 'sum by (%s) (max by (%s, container) (kube_pod_container_resource_requests{%s, resource="%s"}))' % [podKey, podKey, t.matchers, resource];
    local init = 'max by (%s) (kube_pod_init_container_resource_requests{%s, resource="%s"})' % [podKey, t.matchers, resource];
    local effective = 'max by (%s) (label_replace(%s, "source", "containers", "", "") or label_replace(%s, "source", "init", "", ""))' % [podKey, containers, init];
    local requests = 'sum by (cluster, node) (%s * on (cluster, namespace, pod) group_left () %s)' % [effective, t.activePods];
    '100 * (%s or 0 * %s) / %s' % [requests, t.allocatable(resource), t.allocatable(resource)],
  worst(expr):: 'max by (%s) (%s)' % [t.groupBy, expr],
  worstTitle:: if t.groupBy == 'node' then '' else 'of Worst Node ',
};

local nodeStateTimeline = {
  local t = self,
  local key = 'cluster, node',
  local condition(name, status) = '(%s == 1)' % 'max by (%s) (kube_node_status_condition{%s, condition="%s", status="%s"})' % [key, t.matchers, name, status],
  local notReady = '(%s == 0)' % 'max by (%s) (kube_node_status_condition{%s, condition="Ready", status="true"})' % [key, t.matchers],
  local young = '((time() - max by (%s) (kube_node_created{%s})) < 300)' % [key, t.matchers],
  local pressure = 'topk by (%s) (1, label_replace(max by (%s, condition) (kube_node_status_condition{%s, condition=~"MemoryPressure|DiskPressure|PIDPressure|NetworkUnavailable", status="true"}) == 1, "reason", "$1", "condition", "(.*)"))' % [key, key, t.matchers],
  local cordoned = '(%s == 1)' % 'max by (%s) (kube_node_spec_unschedulable{%s})' % [key, t.matchers],
  local labelled(name, expr) = 'label_replace(0 * (%s) + 1, "state", "%s", "", "")' % [expr, name],
  conditions:: {
    Joining: '%s and on (%s) %s' % [notReady, key, young],
    Unreachable: '%s unless on (%s) %s' % [condition('Ready', 'unknown'), key, young],
    NotReady: '%s unless on (%s) %s' % [condition('Ready', 'false'), key, young],
    Pressure: '%s unless on (%s) %s' % [pressure, key, notReady],
    Cordoned: '%s unless on (%s) %s unless on (%s) %s' % [cordoned, key, notReady, key, pressure],
    Ready: '%s unless on (%s) %s unless on (%s) %s' % [condition('Ready', 'true'), key, pressure, key, cordoned],
  },
  states:: ['Joining', 'Unreachable', 'NotReady', 'Pressure', 'Cordoned', 'Ready'],
  unhealthy:: ['Unreachable', 'NotReady', 'Pressure'],
  title:: 'Node State %s' % t.titleGroupBy,
  expr:: 'max by (%s, state, reason) (%s)' % [t.groupBy, std.join(' or ', [labelled(s, t.conditions[s]) for s in t.states])],
  colorBy:: '{{state}}',
  rowBy:: '{{node}}',
  labelBy:: '{{state}}',
  tooltip:: '{{node}} · {{state}} {{reason}} · {{from}} – {{to}}',
  colors:: {
    Ready: '#5b8a5b',
    Joining: '#5a7a94',
    Cordoned: '#7a7a7a',
    Pressure: '#b0902f',
    NotReady: '#a35454',
    Unreachable: '#a35454',
  },
  levels:: {
    root: {
      expr:: 'count by (%s, state) (%s)' % [self.groupBy, std.join(' or ', [labelled(s, self.conditions[s]) for s in self.unhealthy])],
      title:: 'Unhealthy Nodes %s' % self.titleGroupBy,
      rowBy:: '{{%s}}: {{state}}' % self.groupBy,
      labelBy:: '{{value}}',
      tooltip:: '{{%s}} · {{state}}: {{value}} nodes · {{from}} – {{to}}' % self.groupBy,
    },
  },
};

local pvcSeries = {
  local t = self,
  stat(metric):: 'max by (cluster, namespace, persistentvolumeclaim) (%s{%s})' % [metric, t.matchers],
  worst(expr):: 'max by (%s) (%s)' % [t.groupBy, expr],
  worstTitle:: if t.groupBy == 'persistentvolumeclaim' then '' else 'of Worst PVC ',
};

local pvcStateTimeline = {
  local t = self,
  local key = 'cluster, namespace, persistentvolumeclaim',
  local phase(name) = '(max by (%s) (kube_persistentvolumeclaim_status_phase{%s, phase="%s"}) == 1)' % [key, t.matchers, name],
  local young = '((time() - max by (%s) (kube_persistentvolumeclaim_created{%s})) < 300)' % [key, t.matchers],
  local waitsForConsumer = 'max by (%s) (kube_persistentvolumeclaim_info{%s} * on (cluster, storageclass) group_left () max by (cluster, storageclass) (kube_storageclass_info{volume_binding_mode="WaitForFirstConsumer"}))' % [key, t.matchers],
  local resizing = '(max by (%s) (kube_persistentvolumeclaim_status_condition{%s, condition=~"Resizing|FileSystemResizePending", status="true"}) == 1)' % [key, t.matchers],
  local unbound = '(%s unless on (%s) %s)' % [phase('Pending'), key, waitsForConsumer],
  local labelled(name, expr) = 'label_replace(0 * (%s) + 1, "state", "%s", "", "")' % [expr, name],
  conditions:: {
    Lost: phase('Lost'),
    WaitingForConsumer: '%s and on (%s) %s' % [phase('Pending'), key, waitsForConsumer],
    Provisioning: '%s and on (%s) %s' % [unbound, key, young],
    Pending: '%s unless on (%s) %s' % [unbound, key, young],
    Resizing: '%s and on (%s) %s' % [phase('Bound'), key, resizing],
    Bound: '%s unless on (%s) %s' % [phase('Bound'), key, resizing],
  },
  states:: ['Lost', 'WaitingForConsumer', 'Provisioning', 'Pending', 'Resizing', 'Bound'],
  unhealthy:: ['Pending', 'Lost'],
  title:: 'PVC State %s' % t.titleGroupBy,
  expr:: 'max by (%s, state) (%s)' % [t.groupBy, std.join(' or ', [labelled(s, t.conditions[s]) for s in t.states])],
  colorBy:: '{{state}}',
  rowBy:: '{{persistentvolumeclaim}}',
  labelBy:: '{{state}}',
  tooltip:: '{{persistentvolumeclaim}} · {{state}} · {{from}} – {{to}}',
  colors:: {
    Bound: '#5b8a5b',
    Provisioning: '#5a7a94',
    WaitingForConsumer: '#5a7a94',
    Resizing: '#5a7a94',
    Pending: '#b0902f',
    Lost: '#a35454',
  },
  levels:: {
    root: self.context,
    context: {
      expr:: 'count by (%s, state) (%s)' % [self.groupBy, std.join(' or ', [labelled(s, self.conditions[s]) for s in self.unhealthy])],
      title:: 'Unhealthy PVCs %s' % self.titleGroupBy,
      rowBy:: '{{%s}}: {{state}}' % self.groupBy,
      labelBy:: '{{value}}',
      tooltip:: '{{%s}} · {{state}}: {{value}} PVCs · {{from}} – {{to}}' % self.groupBy,
    },
  },
};

{
  a:: telemetry,

  context:: {
    base:: ['telemetry', 'kubernetes'],
    vars:: ['context'],
    labels:: ['cluster'],
    entityPath:: ['kubernetes', '$context'],
    listExpr:: 'count by (%(groupBy)s) (kube_pod_info{%(matchers)s})' % self,
    logs:: {
      all: { expr:: '{cluster="%(context)s"}' % self },
      errors: { expr:: '{cluster="%(context)s"} | detected_level=~"error|critical|fatal"' % self },
    },
    drillDowns:: {},
  },

  namespace:: {
    base:: ['telemetry', 'kubernetes'],
    vars:: ['context', 'namespace'],
    labels:: ['cluster', 'namespace'],
    entityPath:: ['kubernetes', '$context', '$namespace'],
    listExpr:: 'count by (%(groupBy)s) (kube_pod_info{%(matchers)s}) or 0 * count by (%(groupBy)s) (kube_namespace_created{%(matchers)s})' % self,
    logs:: {
      all: { expr:: '{cluster="%(context)s", namespace="%(namespace)s"}' % self },
      errors: { expr:: '{cluster="%(context)s", namespace="%(namespace)s"} | detected_level=~"error|critical|fatal"' % self },
    },
    drillDowns:: {},
  },

  pod:: {
    base:: ['telemetry', 'kubernetes'],
    vars:: ['context', 'namespace', 'pod'],
    labels:: ['cluster', 'namespace', 'pod'],
    entityPath:: ['kubernetes', '$context', '$namespace', '$pod'],
    listExpr:: 'kube_pod_info{%(matchers)s}' % self,
    logs:: {
      all: { expr:: '{cluster="%(context)s", namespace="%(namespace)s"} | pod="%(pod)s"' % self },
      errors: { expr:: '{cluster="%(context)s", namespace="%(namespace)s"} | pod="%(pod)s" | detected_level=~"error|critical|fatal"' % self },
    },
    drillDowns:: {
      state: $.a.promql.stateTimeline.chart + podStateTimeline,
    },
  },

  container:: {
    base:: ['telemetry', 'kubernetes'],
    vars:: ['context', 'namespace', 'pod', 'container'],
    labels:: ['cluster', 'namespace', 'pod', 'container'],
    usageMatchers:: 'image!="", container!="POD"',
    listExpr:: 'kube_pod_container_info{%(matchers)s}' % self,
    logs:: {
      all: { expr:: '{cluster="%(context)s", namespace="%(namespace)s", container="%(container)s"} | pod="%(pod)s"' % self },
      errors: { expr:: '{cluster="%(context)s", namespace="%(namespace)s", container="%(container)s"} | pod="%(pod)s" | detected_level=~"error|critical|fatal"' % self },
    },
    drillDowns:: {
      restarts: $.a.promql.line.chart + containerSeries {
        local t = self,
        local restarts = t.state('increase(kube_pod_container_status_restarts_total{%s}[15m])' % t.matchers),
        local reasons = 'max by (%s, reason) (kube_pod_container_status_last_terminated_reason{%s})' % [t.key, t.matchers],
        expr:: 'sum by (%s, reason) ((%s * on (%s) group_left (reason) %s) or (%s unless on (%s) %s)) > 0' % [t.groupBy, restarts, t.key, reasons, restarts, t.key, reasons],
        legend:: '{{%s}}: {{reason}}' % t.groupBy,
        title:: 'Container Restarts %(titleGroupBy)s(15m)' % self,
      },
      cpu: $.a.promql.line.chart + containerSeries {
        local t = self,
        expr:: 'sum by (%s) (%s)' % [t.groupBy, t.usageRate('container_cpu_usage_seconds_total')],
        title:: 'CPU Usage %(titleGroupBy)s(cores)' % self,
        levels:: {
          container: {
            expr:: t.withBounds(t.usageRate('container_cpu_usage_seconds_total'), 'cpu'),
            legend:: '{{series}}',
          },
        },
      },
      memory: $.a.promql.line.chart + containerSeries {
        local t = self,
        expr:: 'sum by (%s) (%s)' % [t.groupBy, t.usage('container_memory_working_set_bytes')],
        title:: 'Memory Usage %(titleGroupBy)s' % self,
        unit:: 'bytes',
        levels:: {
          container: {
            expr:: t.withBounds(t.usage('container_memory_working_set_bytes'), 'memory'),
            legend:: '{{series}}',
          },
        },
      },
      memoryUtilisation: $.a.promql.line.chart + containerSeries {
        local t = self,
        expr:: t.worst('100 * %s / on (%s) %s' % [t.usage('container_memory_working_set_bytes'), t.key, t.state('kube_pod_container_resource_limits{%s, resource="memory"}' % t.matchers)]),
        title:: 'Memory Utilisation of Limit %(worstTitle)s%(titleGroupBy)s(%%)' % self,
      },
      cpuUtilisation: $.a.promql.line.chart + containerSeries {
        local t = self,
        expr:: t.worst('100 * %s / on (%s) %s' % [t.usageRate('container_cpu_usage_seconds_total'), t.key, t.state('kube_pod_container_resource_requests{%s, resource="cpu"}' % t.matchers)]),
        title:: 'CPU Utilisation of Request %(worstTitle)s%(titleGroupBy)s(%%)' % self,
      },
      throttling: $.a.promql.line.chart + containerSeries {
        local t = self,
        expr:: t.worst('100 * %s / on (%s) %s' % [t.usageRate('container_cpu_cfs_throttled_periods_total'), t.key, t.usageRate('container_cpu_cfs_periods_total')]),
        title:: 'CPU Throttling %(worstTitle)s%(titleGroupBy)s(%% of periods)' % self,
      },
    },
  },

  pvc:: {
    base:: ['telemetry', 'kubernetes'],
    vars:: ['context', 'namespace', 'pvc'],
    entityPath:: ['kubernetes', '$context', '$namespace', '$persistentvolumeclaim'],
    labels:: ['cluster', 'namespace', 'persistentvolumeclaim'],
    titles:: ['Context', 'Namespace', 'PVC'],
    listExpr:: 'kube_persistentvolumeclaim_info{%(matchers)s}' % self,
    drillDowns:: {
      state: $.a.promql.stateTimeline.chart + pvcStateTimeline,
      usage: $.a.promql.line.chart + pvcSeries {
        local t = self,
        expr:: t.worst('100 * %s / %s' % [t.stat('kubelet_volume_stats_used_bytes'), t.stat('kubelet_volume_stats_capacity_bytes')]),
        title:: 'PVC Usage %(worstTitle)s%(titleGroupBy)s(%% of capacity)' % self,
        levels:: {
          pvc: {
            expr:: std.join(' or ', [
              'label_replace(%s, "series", "used", "", "")' % t.stat('kubelet_volume_stats_used_bytes'),
              'label_replace(%s, "series", "capacity", "", "")' % t.stat('kubelet_volume_stats_capacity_bytes'),
              'label_replace(%s, "series", "requested", "", "")' % t.stat('kube_persistentvolumeclaim_resource_requests_storage_bytes'),
            ]),
            legend:: '{{series}}',
            title:: 'PVC Usage',
            unit:: 'bytes',
          },
        },
      },
      inodes: $.a.promql.line.chart + pvcSeries {
        local t = self,
        expr:: t.worst('100 * %s / (%s > 0)' % [t.stat('kubelet_volume_stats_inodes_used'), t.stat('kubelet_volume_stats_inodes')]),
        title:: 'PVC Inodes %(worstTitle)s%(titleGroupBy)s(%% used)' % self,
      },
    },
  },

  deployment:: {
    base:: ['telemetry', 'kubernetes'],
    vars:: ['context', 'namespace', 'deployment'],
    entityPath:: ['kubernetes', '$context', '$namespace', 'apps', '$deployment'],
    labels:: ['cluster', 'namespace', 'deployment'],
    listExpr:: 'kube_deployment_spec_replicas{%(matchers)s}' % self,
    drillDowns:: {
      capacity: $.a.promql.stateTimeline.chart + capacityTimeline + deploymentWorkload,
      rollout: $.a.promql.stateTimeline.chart + deploymentRolloutTimeline,
      pods: $.a.promql.stateTimeline.chart + workloadPodsTimeline + deploymentWorkload,
    },
  },

  statefulset:: {
    base:: ['telemetry', 'kubernetes'],
    vars:: ['context', 'namespace', 'statefulset'],
    entityPath:: ['kubernetes', '$context', '$namespace', 'apps', '$statefulset'],
    labels:: ['cluster', 'namespace', 'statefulset'],
    titles:: ['Context', 'Namespace', 'StatefulSet'],
    listExpr:: 'kube_statefulset_replicas{%(matchers)s}' % self,
    drillDowns:: {
      capacity: $.a.promql.stateTimeline.chart + capacityTimeline + statefulsetWorkload,
      rollout: $.a.promql.stateTimeline.chart + statefulsetRolloutTimeline,
      pods: $.a.promql.stateTimeline.chart + workloadPodsTimeline + statefulsetWorkload,
    },
  },

  daemonset:: {
    base:: ['telemetry', 'kubernetes'],
    vars:: ['context', 'namespace', 'daemonset'],
    entityPath:: ['kubernetes', '$context', '$namespace', 'apps', '$daemonset'],
    labels:: ['cluster', 'namespace', 'daemonset'],
    titles:: ['Context', 'Namespace', 'DaemonSet'],
    listExpr:: 'kube_daemonset_status_desired_number_scheduled{%(matchers)s}' % self,
    drillDowns:: {
      capacity: $.a.promql.stateTimeline.chart + capacityTimeline + daemonsetWorkload,
      rollout: $.a.promql.stateTimeline.chart + daemonsetRolloutTimeline,
      pods: $.a.promql.stateTimeline.chart + workloadPodsTimeline + daemonsetWorkload,
      misscheduled: $.a.promql.line.chart {
        expr:: 'sum by (%(groupBy)s) (max by (cluster, namespace, daemonset) (kube_daemonset_status_number_misscheduled{%(matchers)s})) > 0' % self,
        title:: 'Misscheduled DaemonSet Pods %(titleGroupBy)s' % self,
      },
    },
  },

  node:: {
    base:: ['telemetry', 'kubernetes'],
    vars:: ['context', 'node'],
    labels:: ['cluster', 'node'],
    entityPath:: ['kubernetes', '$context', '$node'],
    nodeExporterInfo:: 'label_replace(max by (cluster, instance, nodename) (node_uname_info), "node", "$1", "nodename", "(.*)")',
    listExpr:: 'count by (%(groupBy)s) (kube_node_info{%(matchers)s})' % self,
    drillDowns:: {
      state: $.a.promql.stateTimeline.chart + nodeStateTimeline,
      cpuRequested: $.a.promql.line.chart + nodeSeries {
        local t = self,
        expr:: t.worst(t.requested('cpu')),
        title:: 'CPU Requested %(worstTitle)s%(titleGroupBy)s(%% of allocatable)' % self,
      },
      memoryRequested: $.a.promql.line.chart + nodeSeries {
        local t = self,
        expr:: t.worst(t.requested('memory')),
        title:: 'Memory Requested %(worstTitle)s%(titleGroupBy)s(%% of allocatable)' % self,
      },
      pods: $.a.promql.line.chart + nodeSeries {
        local t = self,
        local active = 'count by (cluster, node) (max by (cluster, namespace, pod, node) (kube_pod_info{%s}) * on (cluster, namespace, pod) group_left () %s)' % [t.matchers, t.activePods],
        expr:: t.worst('100 * (%s or 0 * %s) / %s' % [active, t.allocatable('pods'), t.allocatable('pods')]),
        title:: 'Pods %(worstTitle)s%(titleGroupBy)s(%% of capacity)' % self,
      },
      cpuUtilisation: $.a.promql.line.chart + nodeSeries {
        local t = self,
        expr:: t.worst(t.fromNodeExporter('100 * (1 - avg by (cluster, instance) (rate(node_cpu_seconds_total{mode="idle"}[5m])))')),
        title:: 'CPU Utilisation %(worstTitle)s%(titleGroupBy)s(%%)' % self,
      },
      memoryUtilisation: $.a.promql.line.chart + nodeSeries {
        local t = self,
        expr:: t.worst(t.fromNodeExporter('100 * (1 - node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes)')),
        title:: 'Memory Utilisation %(worstTitle)s%(titleGroupBy)s(%%)' % self,
      },
      filesystemUtilisation: $.a.promql.line.chart + nodeSeries {
        local t = self,
        local filesystems = 'fstype!~"tmpfs|ramfs|overlay|squashfs|nsfs|fuse.*|nfs.*|cifs|smb.*|ceph.*|glusterfs|9p", mountpoint!~".*/kubelet/pods/.*"',
        local used = '100 * (1 - node_filesystem_avail_bytes{%s} / node_filesystem_size_bytes{%s}) unless node_filesystem_readonly == 1' % [filesystems, filesystems],
        expr:: t.worst(t.fromNodeExporter(used)),
        title:: 'Filesystem Utilisation %(worstTitle)s%(titleGroupBy)s(%% of worst filesystem)' % self,
        levels:: {
          node: {
            expr:: 'max by (mountpoint) (%s)' % t.fromNodeExporter(used, ', mountpoint'),
            legend:: '{{mountpoint}}',
            title:: 'Filesystem Utilisation (%)',
          },
        },
      },
    },
  },

  nodeList: $.a.promql.entities.nodeList([$.context, $.namespace, $.pod, $.container, $.pvc, $.deployment, $.statefulset, $.daemonset, $.node]),
}
