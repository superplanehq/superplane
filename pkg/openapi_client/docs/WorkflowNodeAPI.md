# \WorkflowNodeAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**WorkflowsDeleteNodeQueueItem**](WorkflowNodeAPI.md#WorkflowsDeleteNodeQueueItem) | **Delete** /api/v1/workflows/{workflowId}/nodes/{nodeId}/queue/{itemId} | Delete item from a node&#39;s queue
[**WorkflowsEmitNodeEvent**](WorkflowNodeAPI.md#WorkflowsEmitNodeEvent) | **Post** /api/v1/workflows/{workflowId}/nodes/{nodeId}/events | Emit output event for workflow node
[**WorkflowsInvokeNodeTriggerAction**](WorkflowNodeAPI.md#WorkflowsInvokeNodeTriggerAction) | **Post** /api/v1/workflows/{workflowId}/triggers/{nodeId}/actions/{actionName} | Invoke trigger action
[**WorkflowsListNodeEvents**](WorkflowNodeAPI.md#WorkflowsListNodeEvents) | **Get** /api/v1/workflows/{workflowId}/nodes/{nodeId}/events | List node events
[**WorkflowsListNodeExecutions**](WorkflowNodeAPI.md#WorkflowsListNodeExecutions) | **Get** /api/v1/workflows/{workflowId}/nodes/{nodeId}/executions | List node executions
[**WorkflowsListNodeQueueItems**](WorkflowNodeAPI.md#WorkflowsListNodeQueueItems) | **Get** /api/v1/workflows/{workflowId}/nodes/{nodeId}/queue | List items in a node&#39;s queue
[**WorkflowsUpdateNodePause**](WorkflowNodeAPI.md#WorkflowsUpdateNodePause) | **Patch** /api/v1/workflows/{workflowId}/nodes/{nodeId}/pause | Pause or resume node processing



## WorkflowsDeleteNodeQueueItem

> map[string]interface{} WorkflowsDeleteNodeQueueItem(ctx, workflowId, nodeId, itemId).Execute()

Delete item from a node's queue



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
	nodeId := "nodeId_example" // string | 
	itemId := "itemId_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.WorkflowNodeAPI.WorkflowsDeleteNodeQueueItem(context.Background(), workflowId, nodeId, itemId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowNodeAPI.WorkflowsDeleteNodeQueueItem``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsDeleteNodeQueueItem`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `WorkflowNodeAPI.WorkflowsDeleteNodeQueueItem`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**workflowId** | **string** |  | 
**nodeId** | **string** |  | 
**itemId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsDeleteNodeQueueItemRequest struct via the builder pattern


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


## WorkflowsEmitNodeEvent

> WorkflowsEmitNodeEventResponse WorkflowsEmitNodeEvent(ctx, workflowId, nodeId).Body(body).Execute()

Emit output event for workflow node



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
	nodeId := "nodeId_example" // string | 
	body := *openapiclient.NewWorkflowsEmitNodeEventBody() // WorkflowsEmitNodeEventBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.WorkflowNodeAPI.WorkflowsEmitNodeEvent(context.Background(), workflowId, nodeId).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowNodeAPI.WorkflowsEmitNodeEvent``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsEmitNodeEvent`: WorkflowsEmitNodeEventResponse
	fmt.Fprintf(os.Stdout, "Response from `WorkflowNodeAPI.WorkflowsEmitNodeEvent`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**workflowId** | **string** |  | 
**nodeId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsEmitNodeEventRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | [**WorkflowsEmitNodeEventBody**](WorkflowsEmitNodeEventBody.md) |  | 

### Return type

[**WorkflowsEmitNodeEventResponse**](WorkflowsEmitNodeEventResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## WorkflowsInvokeNodeTriggerAction

> WorkflowsInvokeNodeTriggerActionResponse WorkflowsInvokeNodeTriggerAction(ctx, workflowId, nodeId, actionName).Body(body).Execute()

Invoke trigger action



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
	nodeId := "nodeId_example" // string | 
	actionName := "actionName_example" // string | 
	body := *openapiclient.NewWorkflowsInvokeNodeTriggerActionBody() // WorkflowsInvokeNodeTriggerActionBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.WorkflowNodeAPI.WorkflowsInvokeNodeTriggerAction(context.Background(), workflowId, nodeId, actionName).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowNodeAPI.WorkflowsInvokeNodeTriggerAction``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsInvokeNodeTriggerAction`: WorkflowsInvokeNodeTriggerActionResponse
	fmt.Fprintf(os.Stdout, "Response from `WorkflowNodeAPI.WorkflowsInvokeNodeTriggerAction`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**workflowId** | **string** |  | 
**nodeId** | **string** |  | 
**actionName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsInvokeNodeTriggerActionRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



 **body** | [**WorkflowsInvokeNodeTriggerActionBody**](WorkflowsInvokeNodeTriggerActionBody.md) |  | 

### Return type

[**WorkflowsInvokeNodeTriggerActionResponse**](WorkflowsInvokeNodeTriggerActionResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## WorkflowsListNodeEvents

> WorkflowsListNodeEventsResponse WorkflowsListNodeEvents(ctx, workflowId, nodeId).Limit(limit).Before(before).Execute()

List node events



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
	nodeId := "nodeId_example" // string | 
	limit := int64(789) // int64 |  (optional)
	before := time.Now() // time.Time |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.WorkflowNodeAPI.WorkflowsListNodeEvents(context.Background(), workflowId, nodeId).Limit(limit).Before(before).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowNodeAPI.WorkflowsListNodeEvents``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsListNodeEvents`: WorkflowsListNodeEventsResponse
	fmt.Fprintf(os.Stdout, "Response from `WorkflowNodeAPI.WorkflowsListNodeEvents`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**workflowId** | **string** |  | 
**nodeId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsListNodeEventsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **limit** | **int64** |  | 
 **before** | **time.Time** |  | 

### Return type

[**WorkflowsListNodeEventsResponse**](WorkflowsListNodeEventsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## WorkflowsListNodeExecutions

> WorkflowsListNodeExecutionsResponse WorkflowsListNodeExecutions(ctx, workflowId, nodeId).States(states).Results(results).Limit(limit).Before(before).Execute()

List node executions



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
	nodeId := "nodeId_example" // string | 
	states := []string{"States_example"} // []string |  (optional)
	results := []string{"Results_example"} // []string |  (optional)
	limit := int64(789) // int64 |  (optional)
	before := time.Now() // time.Time |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.WorkflowNodeAPI.WorkflowsListNodeExecutions(context.Background(), workflowId, nodeId).States(states).Results(results).Limit(limit).Before(before).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowNodeAPI.WorkflowsListNodeExecutions``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsListNodeExecutions`: WorkflowsListNodeExecutionsResponse
	fmt.Fprintf(os.Stdout, "Response from `WorkflowNodeAPI.WorkflowsListNodeExecutions`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**workflowId** | **string** |  | 
**nodeId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsListNodeExecutionsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **states** | **[]string** |  | 
 **results** | **[]string** |  | 
 **limit** | **int64** |  | 
 **before** | **time.Time** |  | 

### Return type

[**WorkflowsListNodeExecutionsResponse**](WorkflowsListNodeExecutionsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## WorkflowsListNodeQueueItems

> WorkflowsListNodeQueueItemsResponse WorkflowsListNodeQueueItems(ctx, workflowId, nodeId).Limit(limit).Before(before).Execute()

List items in a node's queue



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
	nodeId := "nodeId_example" // string | 
	limit := int64(789) // int64 |  (optional)
	before := time.Now() // time.Time |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.WorkflowNodeAPI.WorkflowsListNodeQueueItems(context.Background(), workflowId, nodeId).Limit(limit).Before(before).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowNodeAPI.WorkflowsListNodeQueueItems``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsListNodeQueueItems`: WorkflowsListNodeQueueItemsResponse
	fmt.Fprintf(os.Stdout, "Response from `WorkflowNodeAPI.WorkflowsListNodeQueueItems`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**workflowId** | **string** |  | 
**nodeId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsListNodeQueueItemsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **limit** | **int64** |  | 
 **before** | **time.Time** |  | 

### Return type

[**WorkflowsListNodeQueueItemsResponse**](WorkflowsListNodeQueueItemsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## WorkflowsUpdateNodePause

> WorkflowsUpdateNodePauseResponse WorkflowsUpdateNodePause(ctx, workflowId, nodeId).Body(body).Execute()

Pause or resume node processing



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
	nodeId := "nodeId_example" // string | 
	body := *openapiclient.NewWorkflowsUpdateNodePauseBody() // WorkflowsUpdateNodePauseBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.WorkflowNodeAPI.WorkflowsUpdateNodePause(context.Background(), workflowId, nodeId).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowNodeAPI.WorkflowsUpdateNodePause``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsUpdateNodePause`: WorkflowsUpdateNodePauseResponse
	fmt.Fprintf(os.Stdout, "Response from `WorkflowNodeAPI.WorkflowsUpdateNodePause`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**workflowId** | **string** |  | 
**nodeId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsUpdateNodePauseRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | [**WorkflowsUpdateNodePauseBody**](WorkflowsUpdateNodePauseBody.md) |  | 

### Return type

[**WorkflowsUpdateNodePauseResponse**](WorkflowsUpdateNodePauseResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

