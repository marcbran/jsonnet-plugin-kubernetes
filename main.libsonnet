{
  contexts(): std.native('invoke:kubernetes')('contexts', []),
  get(ctx, path): std.native('invoke:kubernetes')('get', [ctx, path]),
}
