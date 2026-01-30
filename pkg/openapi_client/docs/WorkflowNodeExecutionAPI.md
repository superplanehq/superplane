# \WorkflowNodeExecutionAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**WorkflowsCancelExecution**](WorkflowNodeExecutionAPI.md#WorkflowsCancelExecution) | **Patch** /api/v1/workflows/{workflowId}/executions/{executionId}/cancel | Cancel execution
[**WorkflowsInvokeNodeExecutionAction**](WorkflowNodeExecutionAPI.md#WorkflowsInvokeNodeExecutionAction) | **Post** /api/v1/workflows/{workflowId}/executions/{executionId}/actions/{actionName} | Invoke execution action
[**WorkflowsListChildExecutions**](WorkflowNodeExecutionAPI.md#WorkflowsListChildExecutions) | **Post** /api/v1/workflows/{workflowId}/executions/{executionId}/children | List child executions for an execution
[**WorkflowsResolveExecutionErrors**](WorkflowNodeExecutionAPI.md#WorkflowsResolveExecutionErrors) | **Patch** /api/v1/workflows/{workflowId}/executions/resolve | Resolve execution errors



## WorkflowsCancelExecution

> map[string]interface{} WorkflowsCancelExecution(ctx, workflowId, executionId).Body(body).Execute()

Cancel execution



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
	executionId := "executionId_example" // string | 
	body := map[string]interface{}{ ... } // map[string]interface{} | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.WorkflowNodeExecutionAPI.WorkflowsCancelExecution(context.Background(), workflowId, executionId).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowNodeExecutionAPI.WorkflowsCancelExecution``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsCancelExecution`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `WorkflowNodeExecutionAPI.WorkflowsCancelExecution`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**workflowId** | **string** |  | 
**executionId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsCancelExecutionRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | **map[string]interface{}** |  | 

### Return type

**map[string]interface{}**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## WorkflowsInvokeNodeExecutionAction

> map[string]interface{} WorkflowsInvokeNodeExecutionAction(ctx, workflowId, executionId, actionName).Body(body).Execute()

Invoke execution action



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
	executionId := "executionId_example" // string | 
	actionName := "actionName_example" // string | 
	body := *openapiclient.NewWorkflowsInvokeNodeExecutionActionBody() // WorkflowsInvokeNodeExecutionActionBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.WorkflowNodeExecutionAPI.WorkflowsInvokeNodeExecutionAction(context.Background(), workflowId, executionId, actionName).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowNodeExecutionAPI.WorkflowsInvokeNodeExecutionAction``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsInvokeNodeExecutionAction`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `WorkflowNodeExecutionAPI.WorkflowsInvokeNodeExecutionAction`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**workflowId** | **string** |  | 
**executionId** | **string** |  | 
**actionName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsInvokeNodeExecutionActionRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



 **body** | [**WorkflowsInvokeNodeExecutionActionBody**](WorkflowsInvokeNodeExecutionActionBody.md) |  | 

### Return type

**map[string]interface{}**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## WorkflowsListChildExecutions

> WorkflowsListChildExecutionsResponse WorkflowsListChildExecutions(ctx, workflowId, executionId).Body(body).Execute()

List child executions for an execution



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
	executionId := "executionId_example" // string | 
	body := map[string]interface{}{ ... } // map[string]interface{} | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.WorkflowNodeExecutionAPI.WorkflowsListChildExecutions(context.Background(), workflowId, executionId).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowNodeExecutionAPI.WorkflowsListChildExecutions``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsListChildExecutions`: WorkflowsListChildExecutionsResponse
	fmt.Fprintf(os.Stdout, "Response from `WorkflowNodeExecutionAPI.WorkflowsListChildExecutions`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**workflowId** | **string** |  | 
**executionId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsListChildExecutionsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | **map[string]interface{}** |  | 

### Return type

[**WorkflowsListChildExecutionsResponse**](WorkflowsListChildExecutionsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## WorkflowsResolveExecutionErrors

> map[string]interface{} WorkflowsResolveExecutionErrors(ctx, workflowId).Body(body).Execute()

Resolve execution errors



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
	body := *openapiclient.NewWorkflowsResolveExecutionErrorsBody() // WorkflowsResolveExecutionErrorsBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.WorkflowNodeExecutionAPI.WorkflowsResolveExecutionErrors(context.Background(), workflowId).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowNodeExecutionAPI.WorkflowsResolveExecutionErrors``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsResolveExecutionErrors`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `WorkflowNodeExecutionAPI.WorkflowsResolveExecutionErrors`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**workflowId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsResolveExecutionErrorsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**WorkflowsResolveExecutionErrorsBody**](WorkflowsResolveExecutionErrorsBody.md) |  | 

### Return type

**map[string]interface{}**

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

