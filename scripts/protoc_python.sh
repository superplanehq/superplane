#!/bin/bash

set -euo pipefail

# Define Python executable (handles systems where python3 is the standard)
PYTHON_EXE=$(command -v python3 || command -v python)

if [ -z "$PYTHON_EXE" ]; then
    echo "Error: Python is not installed. Please install Python 3 to continue."
    exit 1
fi

echo "--- Generating Python gRPC code ---"

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
PROTO_DIR="$REPO_ROOT/protos"
PYTHON_OUT="$REPO_ROOT/agent/src"

# Verify if grpc_tools is installed before proceeding
if ! $PYTHON_EXE -c "import grpc_tools" &> /dev/null; then
    echo "Error: 'grpcio-tools' is not installed. Run: pip install grpcio-tools"
    exit 1
fi

GRPC_TOOLS_INCLUDE=$($PYTHON_EXE - <<'PY'
import os
import grpc_tools
print(os.path.join(os.path.dirname(grpc_tools.__file__), "_proto"))
PY
)

echo "Target directory: $PYTHON_OUT/private"
mkdir -p "$PYTHON_OUT/private"
touch "$PYTHON_OUT/private/__init__.py"

echo "Compiling protos..."
$PYTHON_EXE -m grpc_tools.protoc \
       --proto_path "$PROTO_DIR" \
       --proto_path "$PROTO_DIR/include" \
       --proto_path "$GRPC_TOOLS_INCLUDE" \
       --python_out="$PYTHON_OUT" \
       "$PROTO_DIR/private/agents.proto" \
       "$PROTO_DIR/usage.proto"

echo "Success: Python code generated in $PYTHON_OUT"