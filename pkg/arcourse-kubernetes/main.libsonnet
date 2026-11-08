local graphLib = import 'graph/main.libsonnet';
local nodeListsAppsLib = import 'nodeLists/apps.libsonnet';
local nodeListsCoreLib = import 'nodeLists/core.libsonnet';
local nodeListsNetworkingK8sIoLib = import 'nodeLists/networking.k8s.io.libsonnet';

{
  graph: graphLib,
  core: nodeListsCoreLib,
  apps: nodeListsAppsLib,
  'networking.k8s.io': nodeListsNetworkingK8sIoLib,
}
