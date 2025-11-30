#!/bin/bash

# Copyright Contributors to the CodeSweep project

set -o errexit
set -o nounset
set -o pipefail

# The root of the build/dist directory
CODESWEEP_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
