# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

terraform {
  version = "TF_VERSION"
}

providers {
  aws         = ["3.18.0"]
  azurerm     = ["2.36.0"]
  google      = ["3.27.0"]
  google-beta = ["3.27.0"]
  openstack   = ["1.28.0"]
  alicloud    = ["1.103.0"]
  packet      = ["2.3.0"]
  template    = ["2.1.2"]
  null        = ["2.1.2"]
}
