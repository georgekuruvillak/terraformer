#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

cd "$(dirname $0)/.."

mkdir -p /tm
/cc/utils/cli.py config attribute --cfg-type kubernetes --cfg-name testmachinery --key kubeconfig > /tm/kubeconfig
# inject the release image tag (content of the VERSION file) as well as the current HEAD commit sha, so that
# 1) TM can retrieve the needed TestDefinitions (from current HEAD)
# 2) we will test the actual release docker image instead of a dev image
/testrunner run \
    --tm-kubeconfig-path=/tm/kubeconfig \
    --no-execution-group \
    --testrun-prefix terraformer-pod-e2e- \
    --timeout=1800 \
    --testruns-chart-path=.ci/testruns/default \
    --set imageTag="$(cat ./VERSION)" \
    --set revision="$(git rev-parse HEAD)"
