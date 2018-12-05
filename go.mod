module github.com/oneconcern/datamon

require (
	cloud.google.com/go v0.33.1
	github.com/aws/aws-sdk-go v1.15.81
	github.com/bmatcuk/doublestar v1.1.1
	github.com/coreos/prometheus-operator v0.25.0 // indirect
	github.com/docker/go-units v0.3.3
	github.com/google/btree v0.0.0-20180813153112-4030bb1f1f0c // indirect
	github.com/google/gofuzz v0.0.0-20170612174753-24818f796faf // indirect
	github.com/googleapis/gax-go v2.0.2+incompatible // indirect
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gosuri/uitable v0.0.0-20160404203958-36ee7e946282 // indirect
	github.com/gregjones/httpcache v0.0.0-20181110185634-c63ab54fda8f // indirect
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/json-iterator/go v1.1.5
	github.com/kubeless/cronjob-trigger v1.0.0 // indirect
	github.com/kubeless/kubeless v1.0.0 // indirect
	github.com/mattn/go-runewidth v0.0.3 // indirect
	github.com/minio/blake2b-simd v0.0.0-20160723061019-3f5f724cb5b1
	github.com/oneconcern/kubeless v1.0.0-alpha.8.0.20181204185124-97e3df54d843
	github.com/oneconcern/pipelines v0.0.0-20181120061409-9184d7135733
	github.com/opentracing/opentracing-go v1.0.2
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/robfig/cron v0.0.0-20180505203441-b41be1df6967 // indirect
	github.com/segmentio/ksuid v1.0.2
	github.com/sirupsen/logrus v1.2.0 // indirect
	github.com/spf13/afero v1.1.2
	github.com/spf13/cobra v0.0.3
	github.com/spf13/viper v1.2.1
	github.com/stretchr/testify v1.2.2
	go.opencensus.io v0.18.0 // indirect
	go.uber.org/zap v1.9.1
	golang.org/x/oauth2 v0.0.0-20181120190819-8f65e3013eba // indirect
	golang.org/x/time v0.0.0-20181108054448-85acf8d2951c // indirect
	google.golang.org/api v0.0.0-20181120235003-faade3cbb06a
	google.golang.org/genproto v0.0.0-20181109154231-b5d43981345b // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.2.1
	k8s.io/api v0.0.0-20181121071145-b7bd5f2d334c // indirect
	k8s.io/apiextensions-apiserver v0.0.0-20181129112646-894efe3a380b // indirect
	k8s.io/apimachinery v0.0.0-20181126191516-4a9a8137c0a1 // indirect
	k8s.io/client-go v9.0.0+incompatible // indirect
	k8s.io/klog v0.1.0 // indirect
	k8s.io/kube-openapi v0.0.0-20181114233023-0317810137be // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)

replace github.com/kubeless/kubeless => github.com/oneconcern/kubeless v1.0.0-alpha.8.0.20181204185124-97e3df54d843
