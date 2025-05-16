#!/bin/bash

<< 'DOCS'
  Generate OpenAPI specs from internal_api definitions.
  This script uses the existing HTTP annotations in the proto file.
DOCS

# When DEBUG env is set - print output of the scripts.
if [[ -e $DEBUG ]];
then
  set -x
fi

OPENAPI_OUT=api/swagger
MODULE_NAME=github.com/superplanehq/superplane
MODULES=$1
PROTO_DIR="protos"

generate_openapi_spec() {
  MODULE=$1
  FILE=$2

  echo "$(bold "Generating OpenAPI spec for $MODULE")"
  
  # Create output directories
  mkdir -p $OPENAPI_OUT

  # Generate gRPC-Gateway code
  protoc --proto_path $PROTO_DIR/ \
         --proto_path $PROTO_DIR/include \
         --openapiv2_out=$OPENAPI_OUT \
         --openapiv2_opt=logtostderr=true \
         --openapiv2_opt=use_go_templates=true \
         $FILE
         
  echo "Generated OpenAPI specification in $OPENAPI_OUT"
}

bold() {
  bold_text=$(tput bold)
  normal_text=$(tput sgr0)
  echo -n "${bold_text}$@${normal_text}"
}

# Main execution
for MODULE in ${MODULES[@]};
do
  generate_openapi_spec $MODULE $PROTO_DIR/$MODULE.proto
done

echo "$(bold "Done generating OpenAPI spec for: ${MODULES[@]}")"