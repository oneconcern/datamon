module github.com/oneconcern/datamon

// NOTE: goleak is a test dependency based on master and not the latest release (stalled)

require (
	cloud.google.com/go v0.49.0
	cloud.google.com/go/storage v1.4.0
	github.com/PuerkitoBio/goquery v1.5.0
	github.com/aws/aws-sdk-go v1.18.6
	github.com/blang/semver v3.5.1+incompatible
	github.com/davecgh/go-spew v1.1.1
	github.com/docker/go-units v0.4.0
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/go-openapi/runtime v0.19.9
	github.com/gobuffalo/envy v1.8.1 // indirect
	github.com/gobuffalo/logger v1.0.3 // indirect
	github.com/gobuffalo/packd v0.3.0
	github.com/gobuffalo/packr/v2 v2.7.1
	github.com/golang/groupcache v0.0.0-20191027212112-611e8accdfc9 // indirect
	github.com/golang/protobuf v1.3.2
	github.com/googleapis/gax-go/v2 v2.0.5
	github.com/hashicorp/go-immutable-radix v1.0.0
	github.com/hashicorp/golang-lru v0.5.3
	github.com/inconshreveable/mousetrap v1.0.0
	github.com/jacobsa/daemonize v0.0.0-20160101105449-e460293e890f
	github.com/jacobsa/fuse v0.0.0-20191026081500-4ee1cf7f62f0
	github.com/jmespath/go-jmespath v0.0.0-20180206201540-c2b33e8439af
	github.com/jstemmer/go-junit-report v0.9.1 // indirect
	github.com/karrick/godirwalk v1.12.0
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd
	github.com/modern-go/reflect2 v1.0.1
	github.com/nightlyone/lockfile v0.0.0-20180618180623-0ad87eef1443
	github.com/opentracing/opentracing-go v1.0.2
	github.com/pelletier/go-toml v1.6.0 // indirect
	github.com/pkg/browser v0.0.0-20180916011732-0a3d74bf9ce4
	github.com/pmezard/go-difflib v1.0.0
	github.com/rhysd/go-github-selfupdate v1.2.1
	github.com/rogpeppe/go-internal v1.5.1
	github.com/segmentio/ksuid v1.0.2
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.5.0
	github.com/stretchr/objx v0.2.0
	github.com/stretchr/testify v1.4.0
	go.opencensus.io v0.22.2
	go.uber.org/atomic v1.5.0
	go.uber.org/goleak v0.10.1-0.20191111212139-7380c5a9fa84
	go.uber.org/multierr v1.3.0
	go.uber.org/zap v1.13.0
	golang.org/x/crypto v0.0.0-20191227163750-53104e6ec876 // indirect
	golang.org/x/exp v0.0.0-20191129062945-2f5052295587 // indirect
	golang.org/x/lint v0.0.0-20191125180803-fdd1cda4f05f // indirect
	golang.org/x/net v0.0.0-20191207000613-e7e4b65ae663
	golang.org/x/oauth2 v0.0.0-20191202225959-858c2ad4c8b6
	golang.org/x/sys v0.0.0-20200107162124-548cf772de50
	golang.org/x/text v0.3.2
	golang.org/x/tools v0.0.0-20200107184032-11e9d9cc0042 // indirect
	google.golang.org/api v0.14.0
	google.golang.org/appengine v1.6.5
	google.golang.org/genproto v0.0.0-20191206224255-0243a4be9c8f
	google.golang.org/grpc v1.25.1
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/yaml.v2 v2.2.7
	gotest.tools v2.2.0+incompatible
	k8s.io/apimachinery v0.17.2
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20200124190032-861946025e34
)

go 1.13
