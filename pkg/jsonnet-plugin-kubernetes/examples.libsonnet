local p = import 'pkg/main.libsonnet';

p.ex({
  context: p.ex([{
    name: 'list pods in all namespaces',
    inputs: ['my-cluster'],
    children: {
      get: p.ex([{
        name: 'list pods',
        inputs: ['/api/v1/pods'],
      }, {
        name: 'list pods in a namespace',
        inputs: ['/api/v1/namespaces/default/pods'],
      }, {
        name: 'get a specific pod',
        inputs: ['/api/v1/namespaces/default/pods/my-pod'],
      }]),
    },
  }]),
})
