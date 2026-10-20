{
  contexts(): std.native('invoke:kubernetes')('contexts', []),
  get(ctx, path, fields=[]): std.native('invoke:kubernetes')('get', [ctx, path, fields]),
  exec(ctx, target, command): std.native('invoke:kubernetes')('exec', [ctx, target, command]) + {
    json:: std.parseJson(self.stdout),
  },
  neat: {
    get(ctx, path, fields=[]): std.native('invoke:kubernetes')('neatGet', [ctx, path, fields]),
  },
}
