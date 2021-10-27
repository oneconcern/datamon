module github.com/oneconcern/datamon

replace github.com/spf13/pflag => github.com/fredbi/pflag v1.0.6-0.20201106154427-e6824c13371a

require (
	cloud.google.com/go/storage v1.18.2
	github.com/PuerkitoBio/goquery v1.5.1
	github.com/aws/aws-sdk-go v1.41.11
	github.com/blang/semver v3.5.1+incompatible
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/cenkalti/backoff/v4 v4.1.1
	github.com/cpuguy83/go-md2man/v2 v2.0.1 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/go-units v0.4.0
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/go-openapi/runtime v0.20.0
	github.com/gobuffalo/logger v1.0.4 // indirect
	github.com/gobuffalo/packd v1.0.0
	github.com/gobuffalo/packr/v2 v2.8.1
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1
	github.com/hashicorp/golang-lru v0.5.4
	github.com/influxdata/influxdb v1.9.5
	github.com/jacobsa/daemonize v0.0.0-20160101105449-e460293e890f
	github.com/jacobsa/fuse v0.0.0-20211019165009-c75d3f26fceb
	github.com/karrick/godirwalk v1.16.1
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/nightlyone/lockfile v1.0.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8
	github.com/rhysd/go-github-selfupdate v1.2.3
	github.com/segmentio/ksuid v1.0.4
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/afero v1.6.0
	github.com/spf13/cobra v1.2.1
	github.com/spf13/viper v1.9.0
	github.com/stretchr/testify v1.7.0
	github.com/ulikunitz/xz v0.5.10 // indirect
	go.opencensus.io v0.23.0
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/goleak v1.1.11-0.20210813005559-691160354723
	go.uber.org/multierr v1.7.0 // indirect
	go.uber.org/zap v1.19.1
	golang.org/x/crypto v0.0.0-20210921155107-089bfa567519 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/net v0.0.0-20211020060615-d418f374d309 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20211025201205-69cdffdb9359
	golang.org/x/term v0.0.0-20210927222741-03fcf44c2211 // indirect
	golang.org/x/text v0.3.7 // indirect
	golang.org/x/tools v0.1.7 // indirect
	google.golang.org/api v0.59.0
	google.golang.org/genproto v0.0.0-20211026145609-4688e4c4e024 // indirect
	google.golang.org/grpc v1.41.0 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
)

go 1.13
