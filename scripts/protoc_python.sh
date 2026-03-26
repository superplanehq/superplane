#!/bin/bash

set -euo pipefail

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
PROTO_DIR="$REPO_ROOT/protos"
PYTHON_OUT="$REPO_ROOT/agent/src"
GRPC_TOOLS_INCLUDE=$(python - <<'PY'
import os
import grpc_tools

print(os.path.join(os.path.dirname(grpc_tools.__file__), "_proto"))
PY
)

mkdir -p "$PYTHON_OUT/private"
touch "$PYTHON_OUT/private/__init__.py"

python -m grpc_tools.protoc \
       --proto_path "$PROTO_DIR" \
       --proto_path "$PROTO_DIR/include" \
       --proto_path "$GRPC_TOOLS_INCLUDE" \
       --python_out="$PYTHON_OUT" \
       "$PROTO_DIR/private/agents.proto"
