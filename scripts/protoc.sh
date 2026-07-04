#!/bin/bash

<< 'DOCS'
  Generate protobuf files from internal_api definitions.
DOCS

# When DEBUG env is set - print output of the scripts.
if [[ -e $DEBUG ]];
then
  set -x
fi

INTERNAL_OUT=pkg/protos
MODULE_NAME=github.com/superplanehq/superplane
MODULES=(${1//,/ })
PROTO_DIR="protos"

generate_proto_definition() {
  FILE=$2

  protoc --proto_path $PROTO_DIR/ \
        --proto_path $PROTO_DIR/include \
        --go-grpc_out=. \
        --go-grpc_opt=module=$MODULE_NAME \
        --go-grpc_opt=require_unimplemented_servers=false \
        --go_out=. \
        --go_opt=module=$MODULE_NAME \
        $FILE
}

generate_proto_files() {
  rm -rf "$INTERNAL_OUT"
  for MODULE in ${MODULES[@]};
  do
    generate_proto_definition $MODULE $PROTO_DIR/$MODULE.proto
  done
}

generate_proto_files