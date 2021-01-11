# SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
#
# SPDX-License-Identifier: Apache-2.0

############# golang-base
FROM eu.gcr.io/gardener-project/3rd/golang:1.15.5 AS golang-base

############# terraform-base
FROM golang-base AS terraform-base

# install unzip (needed for unzipping terraform provider plugins)
RUN apt-get update && \
    apt-get install -y unzip

WORKDIR /tmp/terraformer
COPY ./build/fetch-providers.sh .

# overwrite to build provider-specific image
ARG PROVIDER=all

# copy provider-specifc TF_VERSION
COPY ./build/$PROVIDER/TF_VERSION .

RUN export TF_VERSION=$(cat ./TF_VERSION) && \
    # install terraform and needed provider plugins
    mkdir -p /go/src/github.com/hashicorp && \
    git clone --single-branch --depth 1 --branch v${TF_VERSION} https://github.com/hashicorp/terraform.git /go/src/github.com/hashicorp/terraform && \
    cd /go/src/github.com/hashicorp/terraform && \
    go install ./tools/terraform-bundle

# copy provider-specific terraform-bundle.hcl
COPY ./build/$PROVIDER/terraform-bundle.hcl .
RUN ./fetch-providers.sh

############# builder
FROM golang-base AS builder

WORKDIR /go/src/github.com/gardener/terraformer
COPY . .

ARG PROVIDER=all

RUN make install PROVIDER=$PROVIDER

############# terraformer
FROM eu.gcr.io/gardener-project/3rd/alpine:3.12.3 AS terraformer

RUN apk add --update curl tzdata

WORKDIR /

ENV TF_DEV=true
ENV TF_RELEASE=true

COPY --from=builder /go/bin/terraformer /
COPY --from=terraform-base /tmp/terraformer/terraform /bin/terraform
COPY --from=terraform-base /tmp/terraformer/terraform-provider* /terraform-providers/

ENTRYPOINT ["/terraformer"]

############# dev
FROM golang-base AS dev

WORKDIR /go/src/github.com/gardener/terraformer
VOLUME /go/src/github.com/gardener/terraformer

COPY --from=terraform-base /tmp/terraformer/terraform /bin/terraform
COPY --from=terraform-base /tmp/terraformer/terraform-provider* /terraform-providers/

COPY vendor vendor
COPY Makefile VERSION go.mod go.sum ./

RUN make install-requirements
