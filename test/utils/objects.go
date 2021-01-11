// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"context"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/terraformer/pkg/terraformer"
)

// TestObjects models a set of API objects used in tests
type TestObjects struct {
	ctx    context.Context
	client client.Client

	Namespace              string
	ConfigurationConfigMap *corev1.ConfigMap
	StateConfigMap         *corev1.ConfigMap
	VariablesSecret        *corev1.Secret
}

// PrepareTestObjects creates a default set of needed API objects for tests
func PrepareTestObjects(ctx context.Context, c client.Client, namespacePrefix string) *TestObjects {
	o := &TestObjects{ctx: ctx, client: c}

	if namespacePrefix == "" {
		namespacePrefix = "tf-test-"
	}

	// create test namespace
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: namespacePrefix}}
	Expect(o.client.Create(ctx, ns)).To(Succeed())
	Expect(ns.Name).NotTo(BeEmpty())
	o.Namespace = ns.Name

	var handle CleanupActionHandle
	handle = AddCleanupAction(func() {
		o.CleanupTestObjects(ctx)
		Expect(client.IgnoreNotFound(o.client.Delete(ctx, ns))).To(Succeed())
		RemoveCleanupAction(handle)
	})

	// create configuration ConfigMap
	o.ConfigurationConfigMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "tf-config", Namespace: o.Namespace},
		Data: map[string]string{
			ConfigMainKey: `resource "null_resource" "foo" {
	triggers = {
    some_var = var.SOME_VAR
  }
}`,
			ConfigVarsKey: `variable "SOME_VAR" {
	description = "Some variable"
	type        = string
}`,
		},
	}
	err := o.client.Create(ctx, o.ConfigurationConfigMap)
	Expect(err).NotTo(HaveOccurred())

	// create state ConfigMap
	o.StateConfigMap = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "tf-state", Namespace: o.Namespace},
		Data: map[string]string{
			StateKey: `some state`,
		},
	}
	err = o.client.Create(ctx, o.StateConfigMap)
	Expect(err).NotTo(HaveOccurred())

	// create variables Secret
	o.VariablesSecret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "tf-vars", Namespace: o.Namespace},
		Data: map[string][]byte{
			VarsKey: []byte(`SOME_VAR = "fancy"`),
		},
	}
	err = o.client.Create(ctx, o.VariablesSecret)
	Expect(err).NotTo(HaveOccurred())

	return o
}

// CleanupTestObjects take care to remove the finalizers of the secret and configmaps
func (o *TestObjects) CleanupTestObjects(ctx context.Context) {
	configurationConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: o.ConfigurationConfigMap.Name, Namespace: o.Namespace},
	}
	Expect(client.IgnoreNotFound(o.client.Get(ctx, ObjectKeyFromObject(configurationConfigMap), configurationConfigMap))).To(Succeed())
	copyConfigurationConfigMap := configurationConfigMap.DeepCopy()
	controllerutil.RemoveFinalizer(copyConfigurationConfigMap, terraformer.TerraformerFinalizer)
	Expect(client.IgnoreNotFound(o.client.Patch(ctx, copyConfigurationConfigMap, client.MergeFrom(configurationConfigMap)))).To(Succeed())
	Expect(client.IgnoreNotFound(o.client.Delete(ctx, copyConfigurationConfigMap))).To(Succeed())

	stateConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: o.StateConfigMap.Name, Namespace: o.Namespace},
	}
	Expect(client.IgnoreNotFound(o.client.Get(ctx, ObjectKeyFromObject(stateConfigMap), stateConfigMap))).To(Succeed())
	copyStateConfigMap := stateConfigMap.DeepCopy()
	controllerutil.RemoveFinalizer(copyStateConfigMap, terraformer.TerraformerFinalizer)
	Expect(client.IgnoreNotFound(o.client.Patch(ctx, copyStateConfigMap, client.MergeFrom(stateConfigMap)))).To(Succeed())
	Expect(client.IgnoreNotFound(o.client.Delete(ctx, copyStateConfigMap))).To(Succeed())

	variablesSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: o.VariablesSecret.Name, Namespace: o.Namespace},
	}
	Expect(client.IgnoreNotFound(o.client.Get(ctx, ObjectKeyFromObject(variablesSecret), variablesSecret))).To(Succeed())
	copyVariablesSecret := variablesSecret.DeepCopy()
	controllerutil.RemoveFinalizer(copyVariablesSecret, terraformer.TerraformerFinalizer)
	Expect(client.IgnoreNotFound(o.client.Patch(ctx, copyVariablesSecret, client.MergeFrom(variablesSecret)))).To(Succeed())
	Expect(client.IgnoreNotFound(o.client.Delete(ctx, copyVariablesSecret))).To(Succeed())
}

// Refresh retrieves a fresh copy of the objects from the API server, so that tests can make assertions on them.
func (o *TestObjects) Refresh() {
	Expect(o.client.Get(o.ctx, ObjectKeyFromObject(o.ConfigurationConfigMap), o.ConfigurationConfigMap)).To(Succeed())
	Expect(o.client.Get(o.ctx, ObjectKeyFromObject(o.StateConfigMap), o.StateConfigMap)).To(Succeed())
	Expect(o.client.Get(o.ctx, ObjectKeyFromObject(o.VariablesSecret), o.VariablesSecret)).To(Succeed())
}

// ObjectKeyFromObject returns an ObjectKey for the given object.
func ObjectKeyFromObject(obj metav1.Object) client.ObjectKey {
	return client.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}
}
