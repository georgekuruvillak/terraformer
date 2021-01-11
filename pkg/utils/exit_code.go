// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"fmt"
)

// WithExitCode annotates an error with an exit code.
type WithExitCode struct {
	Code       int
	Underlying error
}

// ExitCode returns the exit code associated with this error.
func (w WithExitCode) ExitCode() int {
	return w.Code
}

// Error implements error.
func (w WithExitCode) Error() string {
	return fmt.Sprintf("terraform command failed with exit code %d: %v", w.Code, w.Underlying)
}

// Unwrap returns the underlying error.
func (w WithExitCode) Unwrap() error {
	return w.Underlying
}
