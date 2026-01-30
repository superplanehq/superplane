# \WorkflowAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**WorkflowsCreateWorkflow**](WorkflowAPI.md#WorkflowsCreateWorkflow) | **Post** /api/v1/workflows | Create workflow
[**WorkflowsDeleteWorkflow**](WorkflowAPI.md#WorkflowsDeleteWorkflow) | **Delete** /api/v1/workflows/{id} | Delete workflow
[**WorkflowsDescribeWorkflow**](WorkflowAPI.md#WorkflowsDescribeWorkflow) | **Get** /api/v1/workflows/{id} | Describe workflow
[**WorkflowsListWorkflows**](WorkflowAPI.md#WorkflowsListWorkflows) | **Get** /api/v1/workflows | List workflows
[**WorkflowsUpdateWorkflow**](WorkflowAPI.md#WorkflowsUpdateWorkflow) | **Put** /api/v1/workflows/{id} | Update workflow



## WorkflowsCreateWorkflow

> WorkflowsCreateWorkflowResponse WorkflowsCreateWorkflow(ctx).Body(body).Execute()

Create workflow



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
	body := *openapiclient.NewWorkflowsCreateWorkflowRequest() // WorkflowsCreateWorkflowRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.WorkflowAPI.WorkflowsCreateWorkflow(context.Background()).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowAPI.WorkflowsCreateWorkflow``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsCreateWorkflow`: WorkflowsCreateWorkflowResponse
	fmt.Fprintf(os.Stdout, "Response from `WorkflowAPI.WorkflowsCreateWorkflow`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsCreateWorkflowRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**WorkflowsCreateWorkflowRequest**](WorkflowsCreateWorkflowRequest.md) |  | 

### Return type

[**WorkflowsCreateWorkflowResponse**](WorkflowsCreateWorkflowResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## WorkflowsDeleteWorkflow

> map[string]interface{} WorkflowsDeleteWorkflow(ctx, id).Execute()

Delete workflow



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
	resp, r, err := apiClient.WorkflowAPI.WorkflowsDeleteWorkflow(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowAPI.WorkflowsDeleteWorkflow``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsDeleteWorkflow`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `WorkflowAPI.WorkflowsDeleteWorkflow`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsDeleteWorkflowRequest struct via the builder pattern


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


## WorkflowsDescribeWorkflow

> WorkflowsDescribeWorkflowResponse WorkflowsDescribeWorkflow(ctx, id).Execute()

Describe workflow



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
	resp, r, err := apiClient.WorkflowAPI.WorkflowsDescribeWorkflow(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowAPI.WorkflowsDescribeWorkflow``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsDescribeWorkflow`: WorkflowsDescribeWorkflowResponse
	fmt.Fprintf(os.Stdout, "Response from `WorkflowAPI.WorkflowsDescribeWorkflow`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsDescribeWorkflowRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**WorkflowsDescribeWorkflowResponse**](WorkflowsDescribeWorkflowResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## WorkflowsListWorkflows

> WorkflowsListWorkflowsResponse WorkflowsListWorkflows(ctx).IncludeTemplates(includeTemplates).Execute()

List workflows



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
	includeTemplates := true // bool |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.WorkflowAPI.WorkflowsListWorkflows(context.Background()).IncludeTemplates(includeTemplates).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowAPI.WorkflowsListWorkflows``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsListWorkflows`: WorkflowsListWorkflowsResponse
	fmt.Fprintf(os.Stdout, "Response from `WorkflowAPI.WorkflowsListWorkflows`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsListWorkflowsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **includeTemplates** | **bool** |  | 

### Return type

[**WorkflowsListWorkflowsResponse**](WorkflowsListWorkflowsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## WorkflowsUpdateWorkflow

> WorkflowsUpdateWorkflowResponse WorkflowsUpdateWorkflow(ctx, id).Body(body).Execute()

Update workflow



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
	body := *openapiclient.NewWorkflowsUpdateWorkflowBody() // WorkflowsUpdateWorkflowBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.WorkflowAPI.WorkflowsUpdateWorkflow(context.Background(), id).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `WorkflowAPI.WorkflowsUpdateWorkflow``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `WorkflowsUpdateWorkflow`: WorkflowsUpdateWorkflowResponse
	fmt.Fprintf(os.Stdout, "Response from `WorkflowAPI.WorkflowsUpdateWorkflow`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiWorkflowsUpdateWorkflowRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**WorkflowsUpdateWorkflowBody**](WorkflowsUpdateWorkflowBody.md) |  | 

### Return type

[**WorkflowsUpdateWorkflowResponse**](WorkflowsUpdateWorkflowResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

