module github.com/oneconcern/datamon

replace github.com/spf13/pflag => github.com/fredbi/pflag v1.0.6-0.20201106154427-e6824c13371a

require (
	cloud.google.com/go/storage v1.23.0
	github.com/PuerkitoBio/goquery v1.8.0
	github.com/aws/aws-sdk-go v1.44.61
	github.com/blang/semver v3.5.1+incompatible
	github.com/cenkalti/backoff/v4 v4.1.3
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/go-units v0.4.0
	github.com/go-chi/chi v4.1.2+incompatible
	github.com/go-openapi/runtime v0.24.1
	github.com/gobuffalo/packd v1.0.1
	github.com/gobuffalo/packr/v2 v2.8.3
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.1
	github.com/hashicorp/golang-lru v0.5.4
	github.com/influxdata/influxdb v1.9.7
	github.com/jacobsa/daemonize v0.0.0-20160101105449-e460293e890f
	github.com/jacobsa/fuse v0.0.0-20220531202254-21122235c77a
	github.com/karrick/godirwalk v1.17.0
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/nightlyone/lockfile v1.0.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pelletier/go-toml/v2 v2.0.2 // indirect
	github.com/pkg/browser v0.0.0-20210911075715-681adbf594b8
	github.com/rhysd/go-github-selfupdate v1.2.3
	github.com/segmentio/ksuid v1.0.4
	github.com/spf13/afero v1.9.1
	github.com/spf13/cobra v1.5.0
	github.com/spf13/viper v1.12.0
	github.com/stretchr/testify v1.8.0
	github.com/subosito/gotenv v1.4.0 // indirect
	github.com/ulikunitz/xz v0.5.10 // indirect
	go.opencensus.io v0.23.0
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/goleak v1.1.12
	go.uber.org/multierr v1.8.0 // indirect
	go.uber.org/zap v1.21.0
	golang.org/x/sync v0.0.0-20220601150217-0de741cfad7f
	golang.org/x/sys v0.0.0-20220624220833-87e55d714810
	golang.org/x/term v0.0.0-20220526004731-065cf7ba2467 // indirect
	golang.org/x/tools v0.1.11 // indirect
	google.golang.org/api v0.87.0
	gopkg.in/ini.v1 v1.66.6 // indirect
	gopkg.in/yaml.v2 v2.4.0
	gotest.tools v2.2.0+incompatible
)

go 1.15
