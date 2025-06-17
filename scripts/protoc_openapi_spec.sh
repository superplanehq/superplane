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
MODULES=(${1//,/ })
PROTO_DIR="protos"
MERGE_FILE_NAME=superplane.swagger

generate_openapi_spec() {
  FILES=$ALL_MODULE_PATHS

  echo "$(bold "Generating OpenAPI spec for $FILES")"
  
  # Create output directories
  mkdir -p $OPENAPI_OUT

  # Generate gRPC-Gateway code
  protoc --proto_path $PROTO_DIR/ \
         --proto_path $PROTO_DIR/include \
         --openapiv2_out=$OPENAPI_OUT \
         --openapiv2_opt=logtostderr=true \
         --openapiv2_opt=use_go_templates=true \
         --openapiv2_opt=allow_merge=true \
         --openapiv2_opt=merge_file_name=$MERGE_FILE_NAME \
         -I . $FILES
         
  echo "Generated OpenAPI specification in $OPENAPI_OUT"
}

bold() {
  bold_text=$(tput bold)
  normal_text=$(tput sgr0)
  echo -n "${bold_text}$@${normal_text}"
}

# Main execution
ALL_MODULE_PATHS=""
for MODULE in ${MODULES[@]};
do
  ALL_MODULE_PATHS+="$PROTO_DIR/$MODULE.proto "
done

generate_openapi_spec $ALL_MODULE_PATHS

echo "$(bold "Done generating OpenAPI spec for: ${MODULES[@]}")"