# vim: set filetype=python :

opts = Variables( 'options.conf', ARGUMENTS )

opts.Add("DESTDIR", 'Set the root directory to install into ( /path/to/DESTDIR )', "")

env = Environment(ENV = {'GOROOT': '/usr/lib/go'},
		  TOOLS=['default', 'go', 'protoc'], options = opts)
env['GO_LIBPATH'] += [env['DESTDIR'] + env['GO_PKGROOT']]
env['GO_GCFLAGS'] = '-I .'
env['GO_LDFLAGS'] = '-L .'

proto_files = env.Protoc([], "geocolo_types.proto",
			 PROTOCFLAGS='--plugin=protoc-gen-go=/usr/lib/go/bin/protoc-gen-go --go_out=.',
			 PROTOCPYTHONOUTDIR='')

geocolo = env.Go('geocolo', ['geolookup_rpc.go', 'geocolo_types.pb.go'])
env.Requires(geocolo, proto_files)
pack = env.GoPack('ancientsolutions.com/geocolo', geocolo)

service = env.Go('geocolo_service', ['geocolo_service.go'])
env.Requires(service, pack)
server = env.GoProgram('geocolo-service', service)

rpcclient = env.Go('geocolo_client', ['geocolo_client.go'])
env.Requires(rpcclient, pack)
client = env.GoProgram('geocolo-client', rpcclient)

env.Install(env['DESTDIR'] + env['GO_PKGROOT'], pack)
env.Install(env['DESTDIR'] + env['ENV']['GOBIN'], server)
env.Install(env['DESTDIR'] + env['ENV']['GOBIN'], client)
env.Alias('install', [env['DESTDIR'] + env['GO_PKGROOT'],
		      env['DESTDIR'] + env['ENV']['GOBIN']])
env.Alias('install-bin', [env['DESTDIR'] + env['ENV']['GOBIN']])
env.Alias('install-libs', [env['DESTDIR'] + env['GO_PKGROOT']])

opts.Save('options.conf', env)
