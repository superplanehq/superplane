# \BlueprintAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**BlueprintsCreateBlueprint**](BlueprintAPI.md#BlueprintsCreateBlueprint) | **Post** /api/v1/blueprints | Create blueprint
[**BlueprintsDeleteBlueprint**](BlueprintAPI.md#BlueprintsDeleteBlueprint) | **Delete** /api/v1/blueprints/{id} | Delete blueprint
[**BlueprintsDescribeBlueprint**](BlueprintAPI.md#BlueprintsDescribeBlueprint) | **Get** /api/v1/blueprints/{id} | Describe blueprint
[**BlueprintsListBlueprints**](BlueprintAPI.md#BlueprintsListBlueprints) | **Get** /api/v1/blueprints | List blueprints
[**BlueprintsUpdateBlueprint**](BlueprintAPI.md#BlueprintsUpdateBlueprint) | **Patch** /api/v1/blueprints/{id} | Update blueprint



## BlueprintsCreateBlueprint

> BlueprintsCreateBlueprintResponse BlueprintsCreateBlueprint(ctx).Body(body).Execute()

Create blueprint



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/openapi_client"
)

func main() {
	body := *openapiclient.NewBlueprintsCreateBlueprintRequest() // BlueprintsCreateBlueprintRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.BlueprintAPI.BlueprintsCreateBlueprint(context.Background()).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `BlueprintAPI.BlueprintsCreateBlueprint``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `BlueprintsCreateBlueprint`: BlueprintsCreateBlueprintResponse
	fmt.Fprintf(os.Stdout, "Response from `BlueprintAPI.BlueprintsCreateBlueprint`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiBlueprintsCreateBlueprintRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**BlueprintsCreateBlueprintRequest**](BlueprintsCreateBlueprintRequest.md) |  | 

### Return type

[**BlueprintsCreateBlueprintResponse**](BlueprintsCreateBlueprintResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## BlueprintsDeleteBlueprint

> map[string]interface{} BlueprintsDeleteBlueprint(ctx, id).Execute()

Delete blueprint



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/openapi_client"
)

func main() {
	id := "id_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.BlueprintAPI.BlueprintsDeleteBlueprint(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `BlueprintAPI.BlueprintsDeleteBlueprint``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `BlueprintsDeleteBlueprint`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `BlueprintAPI.BlueprintsDeleteBlueprint`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiBlueprintsDeleteBlueprintRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

**map[string]interface{}**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## BlueprintsDescribeBlueprint

> BlueprintsDescribeBlueprintResponse BlueprintsDescribeBlueprint(ctx, id).Execute()

Describe blueprint



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/openapi_client"
)

func main() {
	id := "id_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.BlueprintAPI.BlueprintsDescribeBlueprint(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `BlueprintAPI.BlueprintsDescribeBlueprint``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `BlueprintsDescribeBlueprint`: BlueprintsDescribeBlueprintResponse
	fmt.Fprintf(os.Stdout, "Response from `BlueprintAPI.BlueprintsDescribeBlueprint`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiBlueprintsDescribeBlueprintRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**BlueprintsDescribeBlueprintResponse**](BlueprintsDescribeBlueprintResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## BlueprintsListBlueprints

> BlueprintsListBlueprintsResponse BlueprintsListBlueprints(ctx).Execute()

List blueprints



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/openapi_client"
)

func main() {

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.BlueprintAPI.BlueprintsListBlueprints(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `BlueprintAPI.BlueprintsListBlueprints``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `BlueprintsListBlueprints`: BlueprintsListBlueprintsResponse
	fmt.Fprintf(os.Stdout, "Response from `BlueprintAPI.BlueprintsListBlueprints`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiBlueprintsListBlueprintsRequest struct via the builder pattern


### Return type

[**BlueprintsListBlueprintsResponse**](BlueprintsListBlueprintsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## BlueprintsUpdateBlueprint

> BlueprintsUpdateBlueprintResponse BlueprintsUpdateBlueprint(ctx, id).Body(body).Execute()

Update blueprint



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/openapi_client"
)

func main() {
	id := "id_example" // string | 
	body := *openapiclient.NewBlueprintsUpdateBlueprintBody() // BlueprintsUpdateBlueprintBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.BlueprintAPI.BlueprintsUpdateBlueprint(context.Background(), id).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `BlueprintAPI.BlueprintsUpdateBlueprint``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `BlueprintsUpdateBlueprint`: BlueprintsUpdateBlueprintResponse
	fmt.Fprintf(os.Stdout, "Response from `BlueprintAPI.BlueprintsUpdateBlueprint`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiBlueprintsUpdateBlueprintRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**BlueprintsUpdateBlueprintBody**](BlueprintsUpdateBlueprintBody.md) |  | 

### Return type

[**BlueprintsUpdateBlueprintResponse**](BlueprintsUpdateBlueprintResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

