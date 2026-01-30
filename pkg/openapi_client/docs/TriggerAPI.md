# \TriggerAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**TriggersDescribeTrigger**](TriggerAPI.md#TriggersDescribeTrigger) | **Get** /api/v1/triggers/{name} | Describe trigger
[**TriggersListTriggers**](TriggerAPI.md#TriggersListTriggers) | **Get** /api/v1/triggers | List triggers



## TriggersDescribeTrigger

> TriggersDescribeTriggerResponse TriggersDescribeTrigger(ctx, name).Execute()

Describe trigger



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
	resp, r, err := apiClient.TriggerAPI.TriggersDescribeTrigger(context.Background(), name).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `TriggerAPI.TriggersDescribeTrigger``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `TriggersDescribeTrigger`: TriggersDescribeTriggerResponse
	fmt.Fprintf(os.Stdout, "Response from `TriggerAPI.TriggersDescribeTrigger`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**name** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiTriggersDescribeTriggerRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**TriggersDescribeTriggerResponse**](TriggersDescribeTriggerResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## TriggersListTriggers

> TriggersListTriggersResponse TriggersListTriggers(ctx).Execute()

List triggers



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
	resp, r, err := apiClient.TriggerAPI.TriggersListTriggers(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `TriggerAPI.TriggersListTriggers``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `TriggersListTriggers`: TriggersListTriggersResponse
	fmt.Fprintf(os.Stdout, "Response from `TriggerAPI.TriggersListTriggers`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiTriggersListTriggersRequest struct via the builder pattern


### Return type

[**TriggersListTriggersResponse**](TriggersListTriggersResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

