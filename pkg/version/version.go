// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package version

var (
	version Info

	// build time information injected via -ldflags

	gitVersion = "v0.0.0-dev"
	provider   = ""
)

// Info is a struct containing build time information.
type Info struct {
	// GitVersion is the overall codebase version the binary was built from.
	GitVersion string
	// Provider is the container image variant, stating which terraform provider plugins were packaged into the image.
	Provider string
}

func init() {
	version = Info{
		GitVersion: gitVersion,
		Provider:   provider,
	}
}

// Get returns the injected build time information.
func Get() Info {
	return version
}
