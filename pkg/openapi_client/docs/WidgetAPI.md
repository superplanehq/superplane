# \WidgetAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**WidgetsDescribeWidget**](WidgetAPI.md#WidgetsDescribeWidget) | **Get** /api/v1/widgets/{name} | Describe widget
[**WidgetsListWidgets**](WidgetAPI.md#WidgetsListWidgets) | **Get** /api/v1/widgets | List widgets



## WidgetsDescribeWidget

> WidgetsDescribeWidgetResponse WidgetsDescribeWidget(ctx, name).Execute()

Describe widget



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
	resp, r, err := apiClient.WidgetAPI.WidgetsDescribeWidget(context.Background(), name).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WidgetAPI.WidgetsDescribeWidget``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WidgetsDescribeWidget`: WidgetsDescribeWidgetResponse
	fmt.Fprintf(os.Stdout, "Response from `WidgetAPI.WidgetsDescribeWidget`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWidgetsDescribeWidgetRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**WidgetsDescribeWidgetResponse**](WidgetsDescribeWidgetResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## WidgetsListWidgets

> WidgetsListWidgetsResponse WidgetsListWidgets(ctx).Execute()

List widgets



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
	resp, r, err := apiClient.WidgetAPI.WidgetsListWidgets(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WidgetAPI.WidgetsListWidgets``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WidgetsListWidgets`: WidgetsListWidgetsResponse
	fmt.Fprintf(os.Stdout, "Response from `WidgetAPI.WidgetsListWidgets`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiWidgetsListWidgetsRequest struct via the builder pattern


### Return type

[**WidgetsListWidgetsResponse**](WidgetsListWidgetsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

