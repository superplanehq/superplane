# \RolesAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**RolesAssignRole**](RolesAPI.md#RolesAssignRole) | **Post** /api/v1/roles/{roleName}/users | Assign role
[**RolesCreateRole**](RolesAPI.md#RolesCreateRole) | **Post** /api/v1/roles | Create role
[**RolesDeleteRole**](RolesAPI.md#RolesDeleteRole) | **Delete** /api/v1/roles/{roleName} | Delete role
[**RolesDescribeRole**](RolesAPI.md#RolesDescribeRole) | **Get** /api/v1/roles/{roleName} | Describe role
[**RolesListRoles**](RolesAPI.md#RolesListRoles) | **Get** /api/v1/roles | List roles
[**RolesUpdateRole**](RolesAPI.md#RolesUpdateRole) | **Put** /api/v1/roles/{roleName} | Update role



## RolesAssignRole

> map[string]interface{} RolesAssignRole(ctx, roleName).Body(body).Execute()

Assign role



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
	roleName := "roleName_example" // string | 
	body := *openapiclient.NewRolesAssignRoleBody() // RolesAssignRoleBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RolesAPI.RolesAssignRole(context.Background(), roleName).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RolesAPI.RolesAssignRole``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `RolesAssignRole`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `RolesAPI.RolesAssignRole`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**roleName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiRolesAssignRoleRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**RolesAssignRoleBody**](RolesAssignRoleBody.md) |  | 

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


## RolesCreateRole

> RolesCreateRoleResponse RolesCreateRole(ctx).Body(body).Execute()

Create role



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
	body := *openapiclient.NewRolesCreateRoleRequest() // RolesCreateRoleRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RolesAPI.RolesCreateRole(context.Background()).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RolesAPI.RolesCreateRole``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `RolesCreateRole`: RolesCreateRoleResponse
	fmt.Fprintf(os.Stdout, "Response from `RolesAPI.RolesCreateRole`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiRolesCreateRoleRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**RolesCreateRoleRequest**](RolesCreateRoleRequest.md) |  | 

### Return type

[**RolesCreateRoleResponse**](RolesCreateRoleResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## RolesDeleteRole

> map[string]interface{} RolesDeleteRole(ctx, roleName).DomainType(domainType).DomainId(domainId).Execute()

Delete role



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
	roleName := "roleName_example" // string | 
	domainType := "domainType_example" // string |  (optional) (default to "DOMAIN_TYPE_UNSPECIFIED")
	domainId := "domainId_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RolesAPI.RolesDeleteRole(context.Background(), roleName).DomainType(domainType).DomainId(domainId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RolesAPI.RolesDeleteRole``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `RolesDeleteRole`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `RolesAPI.RolesDeleteRole`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**roleName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiRolesDeleteRoleRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **domainType** | **string** |  | [default to &quot;DOMAIN_TYPE_UNSPECIFIED&quot;]
 **domainId** | **string** |  | 

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


## RolesDescribeRole

> RolesDescribeRoleResponse RolesDescribeRole(ctx, roleName).DomainType(domainType).DomainId(domainId).Execute()

Describe role



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
	roleName := "roleName_example" // string | 
	domainType := "domainType_example" // string |  (optional) (default to "DOMAIN_TYPE_UNSPECIFIED")
	domainId := "domainId_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RolesAPI.RolesDescribeRole(context.Background(), roleName).DomainType(domainType).DomainId(domainId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RolesAPI.RolesDescribeRole``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `RolesDescribeRole`: RolesDescribeRoleResponse
	fmt.Fprintf(os.Stdout, "Response from `RolesAPI.RolesDescribeRole`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**roleName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiRolesDescribeRoleRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **domainType** | **string** |  | [default to &quot;DOMAIN_TYPE_UNSPECIFIED&quot;]
 **domainId** | **string** |  | 

### Return type

[**RolesDescribeRoleResponse**](RolesDescribeRoleResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## RolesListRoles

> RolesListRolesResponse RolesListRoles(ctx).DomainType(domainType).DomainId(domainId).Execute()

List roles



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
	domainType := "domainType_example" // string |  (optional) (default to "DOMAIN_TYPE_UNSPECIFIED")
	domainId := "domainId_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RolesAPI.RolesListRoles(context.Background()).DomainType(domainType).DomainId(domainId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RolesAPI.RolesListRoles``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `RolesListRoles`: RolesListRolesResponse
	fmt.Fprintf(os.Stdout, "Response from `RolesAPI.RolesListRoles`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiRolesListRolesRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **domainType** | **string** |  | [default to &quot;DOMAIN_TYPE_UNSPECIFIED&quot;]
 **domainId** | **string** |  | 

### Return type

[**RolesListRolesResponse**](RolesListRolesResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## RolesUpdateRole

> RolesUpdateRoleResponse RolesUpdateRole(ctx, roleName).Body(body).Execute()

Update role



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
	roleName := "roleName_example" // string | 
	body := *openapiclient.NewRolesUpdateRoleBody() // RolesUpdateRoleBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.RolesAPI.RolesUpdateRole(context.Background(), roleName).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `RolesAPI.RolesUpdateRole``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `RolesUpdateRole`: RolesUpdateRoleResponse
	fmt.Fprintf(os.Stdout, "Response from `RolesAPI.RolesUpdateRole`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**roleName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiRolesUpdateRoleRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**RolesUpdateRoleBody**](RolesUpdateRoleBody.md) |  | 

### Return type

[**RolesUpdateRoleResponse**](RolesUpdateRoleResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

