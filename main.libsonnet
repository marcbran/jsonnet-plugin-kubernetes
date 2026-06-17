{
  context(ctx): {
    get(path): std.native('invoke:kubernetes')('request', [{ context: ctx, method: 'GET', path: path }]),
  },
}
