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

# Configuration
OPENAPI_OUT=api/swagger
MODULE_NAME=github.com/superplanehq/superplane
MODULES=(${1//,/ })
PROTO_DIR="protos"

# Output filenames (without .json extension)
MERGE_FILE_NAME=superplane.swagger              # Auto-generated swagger (from protobuf)
MANUAL_SWAGGER_FILE=account-auth.swagger        # Manual swagger (auth/account/setup endpoints)

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

merge_manual_swagger() {
  MANUAL_SWAGGER="$OPENAPI_OUT/$MANUAL_SWAGGER_FILE.json"
  GENERATED_SWAGGER="$OPENAPI_OUT/$MERGE_FILE_NAME.json"
  TEMP_SWAGGER="$OPENAPI_OUT/temp-merged.json"

  # Check if manual swagger exists
  if [ ! -f "$MANUAL_SWAGGER" ]; then
    echo "$(bold "Manual swagger file not found at $MANUAL_SWAGGER, skipping merge")"
    return
  fi

  echo "$(bold "Merging manual swagger with generated swagger")"

  # Use Node.js to merge the swagger files (no additional dependencies needed)
  node scripts/merge-swagger.js "$GENERATED_SWAGGER" "$MANUAL_SWAGGER" "$TEMP_SWAGGER"

  # Replace the generated file with the merged file
  mv "$TEMP_SWAGGER" "$GENERATED_SWAGGER"

  echo "$(bold "Successfully merged manual swagger with generated swagger")"
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

merge_manual_swagger

echo "$(bold "Done generating OpenAPI spec for: ${MODULES[@]}")"