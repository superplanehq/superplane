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
  MODULE=$1
  FILE=$2

  mkdir -p pkg/protos/$MODULE
  protoc --proto_path $PROTO_DIR/ \
        --proto_path $PROTO_DIR/include \
        --go-grpc_out=pkg/protos/$MODULE \
        --go-grpc_opt=paths=source_relative \
        --go-grpc_opt=require_unimplemented_servers=false \
        --go_out=pkg/protos/$MODULE \
        --go_opt=paths=source_relative \
        $FILE
}

generate_proto_files() {
  rm -rf "$INTERNAL_OUT"
  echo "$(bold "Generating proto files")"
  for MODULE in ${MODULES[@]};
  do
    generate_proto_definition $MODULE $PROTO_DIR/$MODULE.proto
  done

  echo "Files generated in $INTERNAL_OUT"
}

bold() {
  bold_text=$(tput bold)
  normal_text=$(tput sgr0)
  echo -n "${bold_text}$@${normal_text}"
}

generate_proto_files
