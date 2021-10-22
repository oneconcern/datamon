module github.com/oneconcern/datamon

replace github.com/spf13/pflag => github.com/fredbi/pflag v1.0.6-0.20201106154427-e6824c13371a

replace github.com/cenkalti/backoff v2.2.1+incompatible => github.com/cenkalti/backoff/v4 v4.1.1

require (
	cloud.google.com/go v0.58.0 // indirect
	cloud.google.com/go/storage v1.9.0
	github.com/PuerkitoBio/goquery v1.5.1
	github.com/aws/aws-sdk-go v1.29.32
	github.com/blang/semver v3.5.1+incompatible
	github.com/cenkalti/backoff v2.2.1+incompatible
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/go-units v0.4.0
	github.com/fsnotify/fsnotify v1.4.9 // indirect
	github.com/go-chi/chi v4.0.4+incompatible
	github.com/go-openapi/runtime v0.19.14
	github.com/gobuffalo/packd v1.0.0
	github.com/gobuffalo/packr/v2 v2.8.0
	github.com/hashicorp/go-immutable-radix v1.2.0
	github.com/hashicorp/golang-lru v0.5.4
	github.com/influxdata/influxdb v1.7.9
	github.com/jacobsa/daemonize v0.0.0-20160101105449-e460293e890f
	github.com/jacobsa/fuse v0.0.0-20200323075136-ffe3eb03daf9
	github.com/jmespath/go-jmespath v0.3.0 // indirect
	github.com/karrick/godirwalk v1.15.5
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/mitchellh/mapstructure v1.2.2 // indirect
	github.com/nightlyone/lockfile v1.0.0
	github.com/onsi/ginkgo v1.10.1 // indirect
	github.com/onsi/gomega v1.7.0 // indirect
	github.com/opentracing/opentracing-go v1.1.0
	github.com/pelletier/go-toml v1.6.0 // indirect
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/rhysd/go-github-selfupdate v1.2.1
	github.com/segmentio/ksuid v1.0.2
	github.com/sirupsen/logrus v1.5.0 // indirect
	github.com/spf13/afero v1.2.2
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.5.1
	go.opencensus.io v0.22.3
	go.uber.org/goleak v1.0.0
	go.uber.org/zap v1.14.1
	golang.org/x/crypto v0.0.0-20200323165209-0ec3e9974c59 // indirect
	golang.org/x/net v0.0.0-20200602114024-627f9648deb9 // indirect
	golang.org/x/sync v0.0.0-20200317015054-43a5402ce75a
	golang.org/x/sys v0.0.0-20200610111108-226ff32320da
	golang.org/x/tools v0.0.0-20200610160956-3e83d1e96d0e // indirect
	google.golang.org/api v0.26.0
	google.golang.org/genproto v0.0.0-20200610104632-a5b850bcf112 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/ini.v1 v1.55.0 // indirect
	gopkg.in/yaml.v2 v2.2.8
	gotest.tools v2.2.0+incompatible
)

go 1.13
