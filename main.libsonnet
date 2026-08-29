{
  contexts(): std.native('invoke:kubernetes')('contexts', []),
  get(ctx, path, fields=[]): std.native('invoke:kubernetes')('get', [ctx, path, fields]),
  neat: {
    get(ctx, path, fields=[]): std.native('invoke:kubernetes')('neatGet', [ctx, path, fields]),
  },
}
