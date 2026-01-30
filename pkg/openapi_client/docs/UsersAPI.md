# \UsersAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**UsersListUserPermissions**](UsersAPI.md#UsersListUserPermissions) | **Get** /api/v1/users/{userId}/permissions | List user permissions
[**UsersListUserRoles**](UsersAPI.md#UsersListUserRoles) | **Get** /api/v1/users/{userId}/roles | Get user roles
[**UsersListUsers**](UsersAPI.md#UsersListUsers) | **Get** /api/v1/users | List users



## UsersListUserPermissions

> UsersListUserPermissionsResponse UsersListUserPermissions(ctx, userId).DomainType(domainType).DomainId(domainId).Execute()

List user permissions



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
	userId := "userId_example" // string | 
	domainType := "domainType_example" // string |  (optional) (default to "DOMAIN_TYPE_UNSPECIFIED")
	domainId := "domainId_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.UsersAPI.UsersListUserPermissions(context.Background(), userId).DomainType(domainType).DomainId(domainId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `UsersAPI.UsersListUserPermissions``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UsersListUserPermissions`: UsersListUserPermissionsResponse
	fmt.Fprintf(os.Stdout, "Response from `UsersAPI.UsersListUserPermissions`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**userId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiUsersListUserPermissionsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **domainType** | **string** |  | [default to &quot;DOMAIN_TYPE_UNSPECIFIED&quot;]
 **domainId** | **string** |  | 

### Return type

[**UsersListUserPermissionsResponse**](UsersListUserPermissionsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UsersListUserRoles

> UsersListUserRolesResponse UsersListUserRoles(ctx, userId).DomainType(domainType).DomainId(domainId).Execute()

Get user roles



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
	userId := "userId_example" // string | 
	domainType := "domainType_example" // string |  (optional) (default to "DOMAIN_TYPE_UNSPECIFIED")
	domainId := "domainId_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.UsersAPI.UsersListUserRoles(context.Background(), userId).DomainType(domainType).DomainId(domainId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `UsersAPI.UsersListUserRoles``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UsersListUserRoles`: UsersListUserRolesResponse
	fmt.Fprintf(os.Stdout, "Response from `UsersAPI.UsersListUserRoles`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**userId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiUsersListUserRolesRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **domainType** | **string** |  | [default to &quot;DOMAIN_TYPE_UNSPECIFIED&quot;]
 **domainId** | **string** |  | 

### Return type

[**UsersListUserRolesResponse**](UsersListUserRolesResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## UsersListUsers

> UsersListUsersResponse UsersListUsers(ctx).DomainType(domainType).DomainId(domainId).Execute()

List users



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
	resp, r, err := apiClient.UsersAPI.UsersListUsers(context.Background()).DomainType(domainType).DomainId(domainId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `UsersAPI.UsersListUsers``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `UsersListUsers`: UsersListUsersResponse
	fmt.Fprintf(os.Stdout, "Response from `UsersAPI.UsersListUsers`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiUsersListUsersRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **domainType** | **string** |  | [default to &quot;DOMAIN_TYPE_UNSPECIFIED&quot;]
 **domainId** | **string** |  | 

### Return type

[**UsersListUsersResponse**](UsersListUsersResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

