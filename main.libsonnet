{
  contexts(): std.native('invoke:kubernetes')('contexts', []),
  get(ctx, path): std.native('invoke:kubernetes')('get', [ctx, path]),
  neat: {
    get(ctx, path): $.get(ctx, path) {
      metadata+: {
        managedFields:: [],
      },
    },
  },
}
