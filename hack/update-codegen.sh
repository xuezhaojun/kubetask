#!/bin/bash

# Copyright Contributors to the CodeSweep project

source "$(dirname "${BASH_SOURCE}")/lib/init.sh"

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
GOPATH="${GOPATH:-$(go env GOPATH)}"
CODEGEN_PKG="${CODEGEN_PKG:-$(go env GOMODCACHE)/k8s.io/code-generator@v0.31.2}"

verify="${VERIFY:-}"

source "${CODEGEN_PKG}/kube_codegen.sh"

kube::codegen::gen_client \
  --output-pkg "github.com/stolostron/codesweep/client" \
  --boilerplate "${SCRIPT_ROOT}/hack/boilerplate.txt" \
  --output-dir ${SCRIPT_ROOT}/client \
  --one-input-api api \
  --with-watch \
  .
