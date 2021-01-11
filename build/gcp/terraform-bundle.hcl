# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

terraform {
  version = "TF_VERSION"
}

providers {
  google      = ["3.27.0"]
  google-beta = ["3.27.0"]
  template    = ["2.1.2"]
  null        = ["2.1.2"]
}
