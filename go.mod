module github.com/gardener/terraformer

go 1.15

require (
	github.com/aws/aws-sdk-go v1.35.26
	github.com/fsnotify/fsnotify v1.4.9
	github.com/gardener/gardener v1.12.4
	github.com/go-logr/logr v0.1.0
	github.com/golang/mock v1.4.4-0.20200731163441-8734ec565a4d
	github.com/google/go-cmp v0.4.1 // indirect
	github.com/hashicorp/go-multierror v1.0.0
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	go.uber.org/zap v1.13.0
	golang.org/x/sys v0.0.0-20200905004654-be1d3432aa8f // indirect
	k8s.io/api v0.18.8
	k8s.io/apimachinery v0.18.8
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/utils v0.0.0-20200619165400-6e3d28b6ed19
	sigs.k8s.io/controller-runtime v0.6.3
)

replace (
	k8s.io/api => k8s.io/api v0.18.8
	k8s.io/apimachinery => k8s.io/apimachinery v0.18.8
	k8s.io/client-go => k8s.io/client-go v0.18.8
)
