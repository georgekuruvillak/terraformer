# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

terraform {
  version = "TF_VERSION"
}

providers {
  alicloud    = ["1.103.0"]
  template    = ["2.1.2"]
  null        = ["2.1.2"]
}
