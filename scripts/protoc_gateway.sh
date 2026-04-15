#!/bin/bash

<< 'DOCS'
  Generate gRPC-Gateway from internal_api definitions.
  This script uses the existing HTTP annotations in the proto file.
DOCS

# When DEBUG env is set - print output of the scripts.
if [[ -e $DEBUG ]];
then
  set -x
fi

INTERNAL_OUT=pkg/protos
GATEWAY_OUT=pkg/protos
MODULE_NAME=github.com/superplanehq/superplane
PROTO_DIR="protos"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# shellcheck source=/dev/null
source "$SCRIPT_DIR/proto_modules.sh"

MODULE_LIST="${1:-$REST_API_MODULES}"
MODULES=(${MODULE_LIST//,/ })

generate_gateway_files() {
  MODULE=$1
  FILE=$2

  echo "$(bold "Generating gRPC-Gateway files for $MODULE")"
  
  # Create output directories
  mkdir -p $GATEWAY_OUT/$MODULE

  # Generate gRPC-Gateway code
  protoc --proto_path $PROTO_DIR/ \
         --proto_path $PROTO_DIR/include \
         --grpc-gateway_out=$GATEWAY_OUT/$MODULE \
         --grpc-gateway_opt=logtostderr=true \
         --grpc-gateway_opt=paths=source_relative \
         $FILE
         
  echo "Generated gRPC-Gateway files in $GATEWAY_OUT/$MODULE"
}

bold() {
  bold_text=$(tput bold)
  normal_text=$(tput sgr0)
  echo -n "${bold_text}$@${normal_text}"
}

# Main execution
for MODULE in ${MODULES[@]};
do
  generate_gateway_files $MODULE $PROTO_DIR/$MODULE.proto
done

echo "$(bold "Done generating gRPC-Gateway for: ${MODULES[@]}")"
