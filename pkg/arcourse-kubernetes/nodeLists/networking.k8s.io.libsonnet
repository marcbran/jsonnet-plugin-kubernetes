local nodeList = [
  [['kubernetes', '$context', '$namespace', 'networking.k8s.io', '$ingress'], {
    linkSpecs+:: [
      {
        at: ['spec', 'rules'],
        keys: [{ const: 'external' }, { path: ['host'] }],
        value: { scheme: 'https', host: [{ path: ['host'] }] },
      },
    ],
  }],
];

{
  nodeList: nodeList,
}
