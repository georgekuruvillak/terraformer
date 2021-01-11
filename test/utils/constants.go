// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

// these consts are replicated from the production to not reuse them in tests
const (
	// ConfigMainKey is the key for the main.tf file
	ConfigMainKey = "main.tf"
	// ConfigVarsKey is the key for the variables.tf file
	ConfigVarsKey = "variables.tf"
	// VarsKey is the key for the terraform.tfvars file
	VarsKey = "terraform.tfvars"
	// StateKey is the key for the terraform.tfstate file
	StateKey = "terraform.tfstate"
)
