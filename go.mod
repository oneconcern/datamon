module github.com/oneconcern/datamon

replace github.com/spf13/pflag => github.com/fredbi/pflag v1.0.6-0.20201106154427-e6824c13371a

require (
	cloud.google.com/go v0.104.0 // indirect
	cloud.google.com/go/compute v1.9.0 // indirect
	cloud.google.com/go/iam v0.4.0 // indirect
	cloud.google.com/go/storage v1.26.0
	github.com/PuerkitoBio/goquery v1.8.0
	github.com/aws/aws-sdk-go v1.44.145
	github.com/blang/semver v3.5.1+incompatible
	github.com/cenkalti/backoff/v4 v4.1.3
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/go-units v0.5.0
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/go-openapi/runtime v0.24.1
	github.com/gobuffalo/packd v1.0.2
	github.com/gobuffalo/packr/v2 v2.8.3
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/googleapis/gax-go/v2 v2.5.1 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1
	github.com/hashicorp/golang-lru v0.5.4
	github.com/influxdata/influxdb v1.10.0
	github.com/jacobsa/daemonize v0.0.0-20160101105449-e460293e890f
	github.com/jacobsa/fuse v0.0.0-20220531202254-21122235c77a
	github.com/karrick/godirwalk v1.17.0
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/nightlyone/lockfile v1.0.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8
	github.com/rhysd/go-github-selfupdate v1.2.3
	github.com/segmentio/ksuid v1.0.4
	github.com/spf13/afero v1.9.2
	github.com/spf13/cobra v1.6.1
	github.com/spf13/viper v1.13.0
	github.com/stretchr/testify v1.8.0
	github.com/ulikunitz/xz v0.5.10 // indirect
	go.opencensus.io v0.23.0
	go.uber.org/atomic v1.10.0 // indirect
	go.uber.org/goleak v1.2.0
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.23.0
	golang.org/x/crypto v0.0.0-20220829220503-c86fa9a7ed90 // indirect
	golang.org/x/oauth2 v0.0.0-20220909003341-f21342109be1 // indirect
	golang.org/x/sync v0.0.0-20220722155255-886fb9371eb4
	golang.org/x/sys v0.1.0
	golang.org/x/xerrors v0.0.0-20220907171357-04be3eba64a2 // indirect
	google.golang.org/api v0.97.0
	google.golang.org/genproto v0.0.0-20220909194730-69f6226f97e5 // indirect
	google.golang.org/grpc v1.49.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
)

go 1.15
