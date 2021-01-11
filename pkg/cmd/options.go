// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/gardener/terraformer/pkg/terraformer"
)

// Options is a struct that holds options for the terraformer binary
type Options struct {
	configurationConfigMapName string
	stateConfigMapName         string
	variablesSecretName        string

	kubeconfig string
	namespace  string

	baseDir string

	completed *terraformer.Config
}

// NewOptions creates new Options
func NewOptions() *Options {
	return &Options{}
}

// Complete tries to complete the provided Options
func (o *Options) Complete() error {
	o.addDefaults()

	if err := o.validate(); err != nil {
		return err
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: o.kubeconfig},
		&clientcmd.ConfigOverrides{Context: clientcmdapi.Context{Namespace: o.namespace}},
	)

	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return err
	}

	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return err
	}

	o.completed = &terraformer.Config{
		ConfigurationConfigMapName: o.configurationConfigMapName,
		StateConfigMapName:         o.stateConfigMapName,
		VariablesSecretName:        o.variablesSecretName,
		Namespace:                  namespace,
		RESTConfig:                 restConfig,
		BaseDir:                    o.baseDir,
	}

	return nil
}

func (o *Options) addDefaults() {
	if len(o.kubeconfig) == 0 {
		o.kubeconfig = os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	}

	if len(o.namespace) == 0 {
		o.namespace = os.Getenv("NAMESPACE")
	}
}

func (o *Options) validate() error {
	if len(o.configurationConfigMapName) == 0 {
		return fmt.Errorf("flag --configuration-configmap-name was not set")
	}
	if len(o.stateConfigMapName) == 0 {
		return fmt.Errorf("flag --state-configmap-name was not set")
	}
	if len(o.variablesSecretName) == 0 {
		return fmt.Errorf("flag --variables-secret-name was not set")
	}

	return nil
}

// AddFlags adds command line flags to a pflag.FlagSet
func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.kubeconfig, clientcmd.RecommendedConfigPathFlag, "", "Path to a kubeconfig. If unset, the KUBECONFIG env var or in-cluster config will be used")
	fs.StringVarP(&o.namespace, "namespace", "n", "", "Namespace to store the configuration resources in. If unset, the NAMESPACE env var or the in-cluster config will be used")
	fs.StringVar(&o.configurationConfigMapName, "configuration-configmap-name", "", "Name of the ConfigMap that holds the main.tf and variables.tf files")
	fs.StringVar(&o.stateConfigMapName, "state-configmap-name", "", "Name of the ConfigMap that the terraform.tfstate file should be stored in")
	fs.StringVar(&o.variablesSecretName, "variables-secret-name", "", "Name of the Secret that holds the terraform.tfvars file")
	fs.StringVar(&o.baseDir, "base-dir", "", "Base directory to be used for all terraform files (defaults to '/')")
}

// Completed returns the completed terraformer.Config
func (o *Options) Completed() *terraformer.Config {
	return o.completed
}
