# \ComponentAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ComponentsDescribeComponent**](ComponentAPI.md#ComponentsDescribeComponent) | **Get** /api/v1/components/{name} | Describe component
[**ComponentsListComponentActions**](ComponentAPI.md#ComponentsListComponentActions) | **Get** /api/v1/components/{name}/actions | List component actions
[**ComponentsListComponents**](ComponentAPI.md#ComponentsListComponents) | **Get** /api/v1/components | List components



## ComponentsDescribeComponent

> ComponentsDescribeComponentResponse ComponentsDescribeComponent(ctx, name).Execute()

Describe component



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
	name := "name_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.ComponentAPI.ComponentsDescribeComponent(context.Background(), name).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ComponentAPI.ComponentsDescribeComponent``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ComponentsDescribeComponent`: ComponentsDescribeComponentResponse
	fmt.Fprintf(os.Stdout, "Response from `ComponentAPI.ComponentsDescribeComponent`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiComponentsDescribeComponentRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**ComponentsDescribeComponentResponse**](ComponentsDescribeComponentResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ComponentsListComponentActions

> ComponentsListComponentActionsResponse ComponentsListComponentActions(ctx, name).Execute()

List component actions



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
	name := "name_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.ComponentAPI.ComponentsListComponentActions(context.Background(), name).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ComponentAPI.ComponentsListComponentActions``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ComponentsListComponentActions`: ComponentsListComponentActionsResponse
	fmt.Fprintf(os.Stdout, "Response from `ComponentAPI.ComponentsListComponentActions`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiComponentsListComponentActionsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**ComponentsListComponentActionsResponse**](ComponentsListComponentActionsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ComponentsListComponents

> ComponentsListComponentsResponse ComponentsListComponents(ctx).Execute()

List components



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
	resp, r, err := apiClient.ComponentAPI.ComponentsListComponents(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `ComponentAPI.ComponentsListComponents``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `ComponentsListComponents`: ComponentsListComponentsResponse
	fmt.Fprintf(os.Stdout, "Response from `ComponentAPI.ComponentsListComponents`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiComponentsListComponentsRequest struct via the builder pattern


### Return type

[**ComponentsListComponentsResponse**](ComponentsListComponentsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

