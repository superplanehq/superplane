# \WorkflowEventAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**WorkflowsListEventExecutions**](WorkflowEventAPI.md#WorkflowsListEventExecutions) | **Get** /api/v1/workflows/{workflowId}/events/{eventId}/executions | List event executions
[**WorkflowsListWorkflowEvents**](WorkflowEventAPI.md#WorkflowsListWorkflowEvents) | **Get** /api/v1/workflows/{workflowId}/events | List workflow events



## WorkflowsListEventExecutions

> WorkflowsListEventExecutionsResponse WorkflowsListEventExecutions(ctx, workflowId, eventId).Execute()

List event executions



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
	workflowId := "workflowId_example" // string | 
	eventId := "eventId_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.WorkflowEventAPI.WorkflowsListEventExecutions(context.Background(), workflowId, eventId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowEventAPI.WorkflowsListEventExecutions``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsListEventExecutions`: WorkflowsListEventExecutionsResponse
	fmt.Fprintf(os.Stdout, "Response from `WorkflowEventAPI.WorkflowsListEventExecutions`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**workflowId** | **string** |  | 
**eventId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsListEventExecutionsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**WorkflowsListEventExecutionsResponse**](WorkflowsListEventExecutionsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## WorkflowsListWorkflowEvents

> WorkflowsListWorkflowEventsResponse WorkflowsListWorkflowEvents(ctx, workflowId).Limit(limit).Before(before).Execute()

List workflow events



### Example

```go
package main

import (
	"context"
	"fmt"
	"os"
    "time"
	openapiclient "github.com/GIT_USER_ID/GIT_REPO_ID/openapi_client"
)

func main() {
	workflowId := "workflowId_example" // string | 
	limit := int64(789) // int64 |  (optional)
	before := time.Now() // time.Time |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.WorkflowEventAPI.WorkflowsListWorkflowEvents(context.Background(), workflowId).Limit(limit).Before(before).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowEventAPI.WorkflowsListWorkflowEvents``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsListWorkflowEvents`: WorkflowsListWorkflowEventsResponse
	fmt.Fprintf(os.Stdout, "Response from `WorkflowEventAPI.WorkflowsListWorkflowEvents`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**workflowId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsListWorkflowEventsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **limit** | **int64** |  | 
 **before** | **time.Time** |  | 

### Return type

[**WorkflowsListWorkflowEventsResponse**](WorkflowsListWorkflowEventsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

