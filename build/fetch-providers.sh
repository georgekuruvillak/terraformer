#!/bin/bash -e
#
# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

terraform-bundle \
  package \
  -os=linux \
  -arch=amd64 \
  <(cat ./terraform-bundle.hcl | sed "s/TF_VERSION/$(cat ./TF_VERSION)/g")

BUNDLE="$(ls -t terraform*.zip | head -1)"
unzip -n $BUNDLE
