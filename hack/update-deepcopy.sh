#!/bin/bash

# Copyright Contributors to the CodeSweep project

source "$(dirname "${BASH_SOURCE}")/lib/init.sh"

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
GOPATH="${GOPATH:-$(go env GOPATH)}"
CODEGEN_PKG="${CODEGEN_PKG:-$(go env GOMODCACHE)/k8s.io/code-generator@v0.31.2}"

source "${CODEGEN_PKG}/kube_codegen.sh"

kube::codegen::gen_helpers \
  --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.txt" \
  api
