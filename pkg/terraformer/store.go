// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package terraformer

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// KeyNotFoundError is returned from a Store.Read if the store does not contain a value for the requested key.
type KeyNotFoundError string

// Error implements error.
func (k KeyNotFoundError) Error() string {
	return fmt.Sprintf("key %q not found", string(k))
}

// Store models storing arbitrary data in a runtime.Object. Implementations have to define how to read and store data
// under a certain key.
type Store interface {
	// Object returns the underlying Object to pass it to a client (for retrieving and updating).
	Object() controllerutil.Object
	// Read returns a reader for reading the contents stored under the given key.
	Read(key string) (io.Reader, error)
	// Store reads the given data and stores it under the given key.
	Store(key string, data io.Reader) error
}

var _ Store = &ConfigMapStore{}

// ConfigMapStore implements Store by storing data in a ConfigMap.
type ConfigMapStore struct {
	*corev1.ConfigMap
}

// Object returns the underlying ConfigMap.
func (c *ConfigMapStore) Object() controllerutil.Object {
	return c.ConfigMap
}

// Read returns a reader for reading the value of the given key in the ConfigMap.
func (c *ConfigMapStore) Read(key string) (io.Reader, error) {
	data, ok := c.Data[key]
	if !ok {
		return nil, KeyNotFoundError(key)
	}

	return strings.NewReader(data), nil
}

// Store reads from the given reader and stores the contents under the given key in the ConfigMap.
func (c *ConfigMapStore) Store(key string, data io.Reader) error {
	if c.ConfigMap.Data == nil {
		c.ConfigMap.Data = make(map[string]string, 1)
	}

	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, data)
	if err != nil {
		return err
	}

	c.ConfigMap.Data[key] = buf.String()
	return nil
}

var _ Store = &SecretStore{}

// SecretStore implements Store by storing data in a Secret.
type SecretStore struct {
	*corev1.Secret
}

// Object returns the underlying Secret.
func (s *SecretStore) Object() controllerutil.Object {
	return s.Secret
}

// Read returns a reader for reading the value of the given key in the Secret.
func (s *SecretStore) Read(key string) (io.Reader, error) {
	data, ok := s.Data[key]
	if !ok {
		return nil, KeyNotFoundError(key)
	}

	return bytes.NewReader(data), nil
}

// Store reads from the given reader and stores the contents under the given key in the Secret.
func (s *SecretStore) Store(key string, data io.Reader) error {
	if s.Secret.Data == nil {
		s.Secret.Data = make(map[string][]byte, 1)
	}

	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, data)
	if err != nil {
		return err
	}

	s.Secret.Data[key] = buf.Bytes()
	return nil
}
