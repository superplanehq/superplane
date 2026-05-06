#!/bin/bash

rm -rf pkg/openapi_client

docker run \
   --rm \
   --user $(id -u):$(id -g) \
  -v ${PWD}:/local \
  openapitools/openapi-generator-cli:v7.13.0 generate \
    -i /local/api/swagger/superplane.swagger.json \
    -g go \
    -o /local/pkg/openapi_client \
    --additional-properties=packageName=openapi_client,enumClassPrefix=true,isGoSubmodule=true,withGoMod=false \
    > /dev/null 2>&1

rm -rf pkg/openapi_client/test
rm -rf pkg/openapi_client/docs
rm -rf pkg/openapi_client/api
rm -rf pkg/openapi_client/.travis.yml
rm -rf pkg/openapi_client/README.md
rm -rf pkg/openapi_client/git_push.sh