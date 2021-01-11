# Terraformer

Terraformer is a tool that can execute Terraform commands (`apply`, `destroy` and `validate`) and can be run as a Pod
inside a Kubernetes cluster.
The Terraform configuration and state files (`main.tf`, `variables.tf`, `terraform.tfvars` and `terraform.tfstate`)
are stored as ConfigMaps and Secrets in the Kubernetes cluster and will be retrieved and updated by Terraformer.

For talking to the API server, Terraformer will either use the service account of its Pod or the provided kubeconfig
(via the `--kubeconfig` flag). Names of the ConfigMaps and Secrets have to be provided via command line flags.
The namespace of the objects can be specified via `--namespace` or will be defaulted to the Pod's namespace.

For more details and example Kubernetes manifests see the [example](example) directory.

**:warning: Running Terraformer as a Job**

Usually, `terraformer apply|destroy|validate` runs within a single Pod. Please note, that running Terraformer as a Job
is not recommended. The Job object will start a new Pod if the first Pod fails or is deleted (for example due to a Node
hardware failure or a reboot). Thus, you may end up in a situation with two running Terraformer Pods at the same time
which can fail with conflicts.

## State file watcher + update worker

While Terraform itself is running, Terraformer watches the state file for changes and updates the state ConfigMap as
soon as a change occurs. Internally, this is achieved by leveraging standard Kubernetes controller mechanisms: the
file watcher inserts an update key into a queue for each file write event, and a separate worker goroutine reads from
the queue and updates the state ConfigMap for every key.

After Terraform exits, Terraformer tries to update the state ConfigMap one last time, and retries the operation with
the exponential backoff until it succeeds or times out.

## Signal handling

Apart from dealing with Terraform configuration and state, Terraformer also handles Pod lifecycle event, i.e. shutdown
signals. It will try to relay `SIGINT` and `SIGTERM` to the running Terraform process in order to stop the ongoing
infrastructure operations on Pod deletion.

## How to run it locally

The `Makefile` specifies targets for running and developing Terraformer locally:

### `make run`

`make run` (with an optional variable `COMMAND=[command]`) will run the given terraformer command (or `apply` by
default) locally on your machine using `go run`. You will have to point Terraformer to a running cluster, where the
config and state is stored, via the `KUBECONFIG` environment variable. This can be any Kubernetes cluster, either a
local one (setup using minikube, kind or the [local nodeless garden](https://github.com/gardener/gardener/blob/master/docs/development/local_setup.md#start-the-gardener))
or even a remote one.

```
$ make run COMMAND=apply
# running `go run ./cmd/terraformer apply`
...
```

### `make start`

`make start` is similar to `make run`, but instead of directly running terraformer on your machine it starts a docker
development container with all the prerequisites installed (like go, terraform and the terraform aws provider plugin)
and starts terraform via `go run` in it.
This allows running terraformer in an environment that is very close to the environment that it is executed in, when
running it in a Kubernetes Pod.

Just as with `make run`, you will have to set your `KUBECONFIG` to point to a Kubernetes cluster used for hosting config
and state. If you are using kind, you might want to use the [example cluster config](example/00-kind-cluster.yaml) to
create a cluster with certificates, that will be trusted from inside the docker VM (on Mac or Windows).

When running this for the first time, the development container will have to be built, which might take a minute.
After that, the image will be reused and only rebuilt if you change something in the files relevant for the image build.

### `make start-dev-container`

The mentioned development container can also be started directly by executing running `make start-dev-container`.
This will open a shell in this container, where you can execute all commands for development and testing like
`make`, `go`, `ginkgo` and so on.

## How to test it

Terraformer currently comes with three suites of tests: a unit test, a binary e2e and a Pod e2e test suite.

### Unit Tests and Binary E2E tests

The unit tests and binary e2e tests can be executed by running `make test`.
Most tests are executed against a local control plane by leveraging the [envtest package](https://github.com/kubernetes-sigs/controller-runtime/tree/master/pkg/envtest).
Therefore, the `kube-apiserver` and `etcd` binaries are fetched to `bin/kubebuilder` and will be started by the tests.

You can also run the tests against a different cluster for debugging or other purposes by setting the 
`USE_EXISTING_CLUSTER` and `KUBECONFIG` environment variable:

```
KUBECONFIG=$HOME/.kube/configs/local/garden.yaml USE_EXISTING_CLUSTER=true make test
```

### Pod E2E tests

The Pod E2E tests can be executed via `make test-e2e`.
This will run e2e tests against a terraformer Pod with the AWS plugin installed.
It uses an existing cluster (given by the `KUBECONFIG` env var) and deploys a terraformer `apply` Pod, that will create
some lightweight resource (ec2 keypair) on AWS.
The test validates, that the resource was created on AWS using the AWS go-sdk and that the state ConfigMap has
been updated accordingly. After that, the test deploys a terraformer `destroy` Pod and validates again the changes
on AWS and the state ConfigMap.

In order to execute this test suite, you will need an existing Kubernetes cluster, where the terraformer Pod will be
deployed (pointed to by the `KUBECONFIG` env var). This can be any Cluster (either local, e.g. via kind) or a remote
one running in the Cloud. Additionally, you will need a set of AWS credentials with which the test resources will be
created (stored under `.kube-secrets/aws/{access_key_id,secret_access_key}.secret`).

```
$ make test-e2e
# Executing pod e2e test with terraformer image eu.gcr.io/gardener-project/gardener/terraformer-aws:v2.1.0-dev-b705a1b4b9bfd47a106998892f48ced0dc8caa56
# If the image for this tag is not built/pushed yet, you have to do so first.
# Or you can use a specific image tag by setting the IMAGE_TAG variable
# like this: `make test-e2e IMAGE_TAG=v2.0.0`
=== RUN   TestTerraformer
Running Suite: Terraformer Pod E2E Suite
========================================
...
```

## Docker Images

Terraformer images are built with every pipeline run and pushed to a public GCR repository.
The list of existing images and tags can be found in [eu.gcr.io/gardener-project/gardener](https://eu.gcr.io/gardener-project/gardener).

### Image variants

This repo features different container image variants, `all` and a few different provider-specific variants.
Each image variant includes the terraformer binary itself, plus terraform and some terraform provider plugins.

The image variants are specified under `build`, each directory represents one variant.
Each variant defines which version of terraform (`TF_VERSION` file) and which terraform provider plugins
(with their respective versions, see (`terraform-bundle.hcl` file)) should be packaged into the image as well.

Historically, terraformer images included provider plugins for all Gardener provider extensions that were using
terraformer for managing a Shoot cluster's infrastructure. This image is equivalent to the `all` variant.
Packaging all provider plugins makes terraformer's images quite large and thus unnecessarily increases image pull time,
network traffic and cost.
With the different image variants, Gardener provider extensions can now deploy terraformer images with only the needed
plugins inside. Also, the different extensions don't have to agree on a common terraform version, but are able to choose
the terraform version which they want to use in their provider-specific image.

The `all` image variant is tagged as `eu.gcr.io/gardener-project/gardener/terraformer`, while the provider-specific
image variants are tagged as `eu.gcr.io/gardener-project/gardener/terraformer-{aws,gcp,...}`.

### Building images locally

You can use `make docker-images` to build all image variants locally.

Alternatively, `make docker-image PROVIDER={all,aws,gcp,...}` can be used to build only one specific image variant.

## Find out more & Get in touch!

Terraformer was presented in the Gardener Community Call on Nov, 13th 2020.  
Watch the [recording](https://youtu.be/4sQs_Hj6xpY) to learn the story behind terraformer v2 and how it is used
in our provider extensions!

Feedback and contributions are always welcome!

Reach out to us:
- via our [Slack channel](https://kubernetes.slack.com/messages/gardener), get an invite [here](http://slack.k8s.io/)
- on our [Mailing List](https://groups.google.com/forum/?fromgroups#!forum/gardener)
- on Twitter [@GardenerProject](https://twitter.com/GardenerProject)
- in our bi-weekly [Community Call](https://github.com/gardener/documentation/blob/master/CONTRIBUTING.md#weekly-meeting)

Please find further resources about our project here:

- [Our landing page gardener.cloud](https://gardener.cloud/)
- ["Gardener Project Update" blog on kubernetes.io](https://kubernetes.io/blog/2019/12/02/gardener-project-update/).
- ["Gardener, the Kubernetes Botanist" blog on kubernetes.io](https://kubernetes.io/blog/2018/05/17/gardener/)
- [SAP news article about "Project Gardener"](https://news.sap.com/2018/11/hasso-plattner-founders-award-finalist-profile-project-gardener/)
- [Introduction movie: "Gardener - Planting the Seeds of Success in the Cloud"](https://www.sap-tv.com/video/40962/gardener-planting-the-seeds-of-success-in-the-cloud)
- ["Thinking Cloud Native" talk at EclipseCon 2018](https://www.youtube.com/watch?v=bfw22WPg99A)
- [Blog - "Showcase of Gardener at OSCON 2018"](https://blogs.sap.com/2018/07/26/showcase-of-gardener-at-oscon/)
