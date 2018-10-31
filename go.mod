module github.com/oneconcern/datamon

require (
	cloud.google.com/go v0.31.0 // indirect
	github.com/AndreasBriese/bbloom v0.0.0-20180913140656-343706a395b7
	github.com/aws/aws-sdk-go v1.15.56
	github.com/beorn7/perks v0.0.0-20180321164747-3a771d992973
	github.com/codahale/hdrhistogram v0.0.0-20161010025455-3a0bb77429bd
	github.com/coreos/prometheus-operator v0.25.0 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/dgraph-io/badger v1.5.3
	github.com/dgryski/go-farm v0.0.0-20180109070241-2de33835d102
	github.com/docker/go-units v0.3.3
	github.com/fatih/color v1.7.0
	github.com/felixge/httpsnoop v1.0.0
	github.com/fsnotify/fsnotify v1.4.7
	github.com/ghodss/yaml v1.0.0
	github.com/go-ini/ini v1.38.2
	github.com/go-openapi/swag v0.0.0-20180715190254-becd2f08beaf
	github.com/gogo/protobuf v1.1.1
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.2.0
	github.com/google/btree v0.0.0-20180813153112-4030bb1f1f0c // indirect
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf
	github.com/googleapis/gnostic v0.2.0
	github.com/gorilla/websocket v1.4.0
	github.com/gosuri/uitable v0.0.0-20160404203958-36ee7e946282
	github.com/gregjones/httpcache v0.0.0-20180305231024-9cad4c3443a7 // indirect
	github.com/hashicorp/hcl v1.0.0
	github.com/howeyc/gopass v0.0.0-20170109162249-bf9dde6d0d2c
	github.com/imdario/mergo v0.3.6
	github.com/inconshreveable/mousetrap v1.0.0
	github.com/jmespath/go-jmespath v0.0.0-20160202185014-0b12d6b521d8
	github.com/json-iterator/go v1.1.5
	github.com/kardianos/osext v0.0.0-20170510131534-ae77be60afb1
	github.com/kubeless/kubeless v0.6.0
	github.com/magiconair/properties v1.8.0
	github.com/mailru/easyjson v0.0.0-20180823135443-60711f1a8329
	github.com/mattn/go-colorable v0.0.9
	github.com/mattn/go-isatty v0.0.4
	github.com/mattn/go-runewidth v0.0.3
	github.com/matttproud/golang_protobuf_extensions v1.0.1
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/mitchellh/mapstructure v1.0.0
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd
	github.com/modern-go/reflect2 v0.0.0-20180701023420-4b7aa43c6742
	github.com/nats-io/go-nats v1.6.0
	github.com/nats-io/go-nats-streaming v0.3.4
	github.com/nats-io/nuid v1.0.0
	github.com/oneconcern/pipelines v0.0.0-20180813230703-c522c69bbdb9
	github.com/opentracing-contrib/go-stdlib v0.0.0-20180702182724-07a764486eb1 // indirect
	github.com/opentracing/opentracing-go v1.0.2
	github.com/pelletier/go-toml v1.2.0
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pkg/errors v0.8.0
	github.com/pmezard/go-difflib v1.0.0
	github.com/prometheus/client_golang v0.8.0
	github.com/prometheus/client_model v0.0.0-20180712105110-5c3871d89910
	github.com/prometheus/common v0.0.0-20180801064454-c7de2306084e
	github.com/prometheus/procfs v0.0.0-20180725123919-05ee40e3a273
	github.com/sirupsen/logrus v1.1.1 // indirect
	github.com/spf13/afero v1.1.2
	github.com/spf13/cast v1.2.0
	github.com/spf13/cobra v0.0.3
	github.com/spf13/jwalterweatherman v1.0.0
	github.com/spf13/pflag v1.0.2
	github.com/spf13/viper v1.2.1
	github.com/stretchr/testify v1.2.2
	github.com/uber/jaeger-client-go v2.14.0+incompatible
	github.com/uber/jaeger-lib v1.5.0
	github.com/vektah/gqlgen v0.0.0-20180714070128-381b34691fd9
	go.uber.org/atomic v1.3.2
	go.uber.org/multierr v1.1.0
	go.uber.org/zap v1.9.1
	golang.org/x/crypto v0.0.0-20180910181607-0e37d006457b
	golang.org/x/net v0.0.0-20180911220305-26e67e76b6c3
	golang.org/x/oauth2 v0.0.0-20181031022657-8527f56f7107 // indirect
	golang.org/x/sys v0.0.0-20180909124046-d0be0721c37e
	golang.org/x/text v0.3.0
	golang.org/x/time v0.0.0-20180412165947-fbb02b2291d2
	gopkg.in/inf.v0 v0.9.1
	gopkg.in/yaml.v2 v2.2.1
	k8s.io/api v0.0.0-20180913155108-f456898a08e4
	k8s.io/apiextensions-apiserver v0.0.0-20181031052806-bb9932e89b5b // indirect
	k8s.io/apimachinery v0.0.0-20181031012033-2e0dc82819fd
	k8s.io/client-go v9.0.0+incompatible
	k8s.io/kube-openapi v0.0.0-20181026222903-0d1aeffe1c68 // indirect
)

replace github.com/spf13/cobra => github.com/babysnakes/cobra v0.0.2-0.20180603190830-61ca3af7ef22

replace github.com/opentracing-contrib/go-stdlib => github.com/casualjim/go-stdlib v0.0.0-20180812144825-f21c79781714
