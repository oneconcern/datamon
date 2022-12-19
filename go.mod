module github.com/oneconcern/datamon

replace github.com/spf13/pflag => github.com/fredbi/pflag v1.0.6-0.20201106154427-e6824c13371a

require (
	cloud.google.com/go v0.107.0 // indirect
	cloud.google.com/go/storage v1.28.0
	github.com/PuerkitoBio/goquery v1.8.0
	github.com/aws/aws-sdk-go v1.44.149
	github.com/blang/semver v3.5.1+incompatible
	github.com/cenkalti/backoff/v4 v4.2.0
	github.com/davecgh/go-spew v1.1.1
	github.com/dgraph-io/badger/v3 v3.2103.4
	github.com/docker/go-units v0.5.0
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/go-openapi/runtime v0.25.0
	github.com/gobuffalo/packd v1.0.2
	github.com/gobuffalo/packr/v2 v2.8.3
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1
	github.com/hashicorp/golang-lru v0.6.0
	github.com/influxdata/influxdb v1.10.0
	github.com/jacobsa/daemonize v0.0.0-20160101105449-e460293e890f
	github.com/jacobsa/fuse v0.0.0-20220531202254-21122235c77a
	github.com/karrick/godirwalk v1.17.0
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/nightlyone/lockfile v1.0.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pelletier/go-toml/v2 v2.0.6 // indirect
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8
	github.com/rhysd/go-github-selfupdate v1.2.3
	github.com/segmentio/ksuid v1.0.4
	github.com/spf13/afero v1.9.3
	github.com/spf13/cobra v1.6.1
	github.com/spf13/viper v1.14.0
	github.com/stretchr/testify v1.8.1
	github.com/ulikunitz/xz v0.5.10 // indirect
	go.opencensus.io v0.24.0
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/goleak v1.2.0
	go.uber.org/zap v1.24.0
	golang.org/x/crypto v0.0.0-20220829220503-c86fa9a7ed90 // indirect
	golang.org/x/oauth2 v0.2.0 // indirect
	golang.org/x/sync v0.1.0
	golang.org/x/sys v0.2.0
	google.golang.org/api v0.103.0
	google.golang.org/genproto v0.0.0-20221118155620-16455021b5e6 // indirect
	google.golang.org/grpc v1.51.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
)

go 1.15
