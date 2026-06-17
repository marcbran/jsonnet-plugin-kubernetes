{
  contexts(): std.native('invoke:kubernetes')('contexts', []),
  context(ctx): {
    get(path): std.native('invoke:kubernetes')('request', [{ context: ctx, method: 'GET', path: path }]),
  },
}
