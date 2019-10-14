module github.com/oneconcern/datamon

replace google.golang.org/api => github.com/googleapis/google-api-go-client v0.2.1-0.20190318183801-2dc3ad4d67ba

replace github.com/spf13/cobra => github.com/babysnakes/cobra v0.0.2-0.20180603190830-61ca3af7ef22

replace github.com/spf13/pflag => github.com/spf13/pflag v1.0.3

replace go.uber.org/goleak => go.uber.org/goleak v0.10.1-0.20190823232112-227bd74c3482

require (
	cloud.google.com/go v0.37.1
	github.com/aws/aws-sdk-go v1.18.6
	github.com/container-storage-interface/spec v0.3.0
	github.com/coreos/go-etcd v2.0.0+incompatible // indirect
	github.com/docker/go-units v0.3.3
	github.com/go-chi/chi v4.0.2+incompatible
	github.com/gobuffalo/envy v1.7.1 // indirect
	github.com/gobuffalo/logger v1.0.1 // indirect
	github.com/gobuffalo/packd v0.3.0
	github.com/gobuffalo/packr/v2 v2.6.0
	github.com/gogo/protobuf v1.3.0 // indirect
	github.com/golang/mock v1.3.1 // indirect
	github.com/golangci/go-tools v0.0.0-20190318055746-e32c54105b7c // indirect
	github.com/golangci/gocyclo v0.0.0-20180528144436-0a533e8fa43d // indirect
	github.com/golangci/gofmt v0.0.0-20190930125516-244bba706f1a // indirect
	github.com/golangci/golangci-lint v1.19.1 // indirect
	github.com/golangci/gosec v0.0.0-20190211064107-66fb7fc33547 // indirect
	github.com/golangci/revgrep v0.0.0-20180812185044-276a5c0a1039 // indirect
	github.com/gostaticanalysis/analysisutil v0.0.3 // indirect
	github.com/hashicorp/go-immutable-radix v1.0.0
	github.com/hashicorp/golang-lru v0.5.0
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/jacobsa/daemonize v0.0.0-20160101105449-e460293e890f
	github.com/jacobsa/fuse v0.0.0-20180417054321-cd3959611bcb
	github.com/json-iterator/go v1.1.6
	github.com/karrick/godirwalk v1.10.12
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/markbates/oncer v0.0.0-20181203154359-bf2de49a0be2 // indirect
	github.com/markbates/safe v1.0.1 // indirect
	github.com/matoous/godox v0.0.0-20190911065817-5d6d842e92eb // indirect
	github.com/mattn/go-colorable v0.1.4 // indirect
	github.com/mattn/go-isatty v0.0.9 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/nightlyone/lockfile v0.0.0-20180618180623-0ad87eef1443
	github.com/opentracing/opentracing-go v1.0.2
	github.com/pelletier/go-toml v1.4.0 // indirect
	github.com/securego/gosec v0.0.0-20191002120514-e680875ea14d // indirect
	github.com/segmentio/ksuid v1.0.2
	github.com/spf13/afero v1.2.2
	github.com/spf13/cobra v0.0.5
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/viper v1.4.0
	github.com/stretchr/testify v1.4.0
	github.com/timakin/bodyclose v0.0.0-20190930140734-f7f2e9bca95e // indirect
	github.com/ugorji/go/codec v0.0.0-20181204163529-d75b2dcb6bc8 // indirect
	github.com/ultraware/whitespace v0.0.4 // indirect
	go.uber.org/goleak v0.0.0-00010101000000-000000000000
	go.uber.org/zap v1.10.0
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
	golang.org/x/sys v0.0.0-20191003212358-c178f38b412c
	golang.org/x/tools v0.0.0-20191004211743-43d3a2ca2ae9
	google.golang.org/api v0.2.0
	google.golang.org/grpc v1.21.0
	gopkg.in/yaml.v2 v2.2.4
	gotest.tools/gotestsum v0.3.5 // indirect
	k8s.io/apimachinery v0.0.0-20190531161113-d9689afd32c1 // indirect
	k8s.io/kubernetes v1.14.2
	k8s.io/utils v0.0.0-20190520173318-324c5df7d3f0 // indirect
	mvdan.cc/unparam v0.0.0-20190917161559-b83a221c10a2 // indirect
	sourcegraph.com/sqs/pbtypes v1.0.0 // indirect
)

go 1.13
