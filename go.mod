module github.com/oneconcern/datamon

replace google.golang.org/api => github.com/googleapis/google-api-go-client v0.2.1-0.20190318183801-2dc3ad4d67ba

replace github.com/spf13/cobra => github.com/babysnakes/cobra v0.0.2-0.20180603190830-61ca3af7ef22

replace github.com/spf13/pflag => github.com/spf13/pflag v1.0.3

replace go.uber.org/goleak => go.uber.org/goleak v0.10.1-0.20190823232112-227bd74c3482

require (
	cloud.google.com/go v0.37.1
	github.com/aws/aws-sdk-go v1.18.6
	github.com/docker/go-units v0.3.3
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/gobuffalo/packd v0.3.0
	github.com/gobuffalo/packr/v2 v2.7.1
	github.com/golang/protobuf v1.3.2 // indirect
	github.com/google/go-cmp v0.3.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.0.0
	github.com/hashicorp/golang-lru v0.5.0
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jacobsa/daemonize v0.0.0-20160101105449-e460293e890f
	github.com/jacobsa/fuse v0.0.0-20180417054321-cd3959611bcb
	github.com/karrick/godirwalk v1.12.0
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/nightlyone/lockfile v0.0.0-20180618180623-0ad87eef1443
	github.com/opentracing/opentracing-go v1.0.2
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/pkg/errors v0.8.1 // indirect
	github.com/rogpeppe/go-internal v1.5.0 // indirect
	github.com/segmentio/ksuid v1.0.2
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.4.0
	github.com/stretchr/objx v0.2.0 // indirect
	github.com/stretchr/testify v1.4.0
	go.uber.org/goleak v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.10.0
	golang.org/x/crypto v0.0.0-20191011191535-87dc89f01550 // indirect
	golang.org/x/net v0.0.0-20190923162816-aa69164e4478 // indirect
	golang.org/x/sys v0.0.0-20191023151326-f89234f9a2c2
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/api v0.2.0
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.2.4
)

go 1.13
