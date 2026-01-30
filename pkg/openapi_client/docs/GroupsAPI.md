# \GroupsAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GroupsAddUserToGroup**](GroupsAPI.md#GroupsAddUserToGroup) | **Post** /api/v1/groups/{groupName}/users | Add user to group
[**GroupsCreateGroup**](GroupsAPI.md#GroupsCreateGroup) | **Post** /api/v1/groups | Create group
[**GroupsDeleteGroup**](GroupsAPI.md#GroupsDeleteGroup) | **Delete** /api/v1/groups/{groupName} | Delete group
[**GroupsDescribeGroup**](GroupsAPI.md#GroupsDescribeGroup) | **Get** /api/v1/groups/{groupName} | Get group
[**GroupsListGroupUsers**](GroupsAPI.md#GroupsListGroupUsers) | **Get** /api/v1/groups/{groupName}/users | Get group users
[**GroupsListGroups**](GroupsAPI.md#GroupsListGroups) | **Get** /api/v1/groups | List groups
[**GroupsRemoveUserFromGroup**](GroupsAPI.md#GroupsRemoveUserFromGroup) | **Patch** /api/v1/groups/{groupName}/users/remove | Remove user from group
[**GroupsUpdateGroup**](GroupsAPI.md#GroupsUpdateGroup) | **Put** /api/v1/groups/{groupName} | Update group



## GroupsAddUserToGroup

> map[string]interface{} GroupsAddUserToGroup(ctx, groupName).Body(body).Execute()

Add user to group



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
	groupName := "groupName_example" // string | 
	body := *openapiclient.NewGroupsAddUserToGroupBody() // GroupsAddUserToGroupBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.GroupsAPI.GroupsAddUserToGroup(context.Background(), groupName).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `GroupsAPI.GroupsAddUserToGroup``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GroupsAddUserToGroup`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `GroupsAPI.GroupsAddUserToGroup`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**groupName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiGroupsAddUserToGroupRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**GroupsAddUserToGroupBody**](GroupsAddUserToGroupBody.md) |  | 

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


## GroupsCreateGroup

> GroupsCreateGroupResponse GroupsCreateGroup(ctx).Body(body).Execute()

Create group



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
	body := *openapiclient.NewGroupsCreateGroupRequest() // GroupsCreateGroupRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.GroupsAPI.GroupsCreateGroup(context.Background()).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `GroupsAPI.GroupsCreateGroup``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GroupsCreateGroup`: GroupsCreateGroupResponse
	fmt.Fprintf(os.Stdout, "Response from `GroupsAPI.GroupsCreateGroup`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGroupsCreateGroupRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**GroupsCreateGroupRequest**](GroupsCreateGroupRequest.md) |  | 

### Return type

[**GroupsCreateGroupResponse**](GroupsCreateGroupResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GroupsDeleteGroup

> map[string]interface{} GroupsDeleteGroup(ctx, groupName).DomainType(domainType).DomainId(domainId).Execute()

Delete group



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
	groupName := "groupName_example" // string | 
	domainType := "domainType_example" // string |  (optional) (default to "DOMAIN_TYPE_UNSPECIFIED")
	domainId := "domainId_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.GroupsAPI.GroupsDeleteGroup(context.Background(), groupName).DomainType(domainType).DomainId(domainId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `GroupsAPI.GroupsDeleteGroup``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GroupsDeleteGroup`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `GroupsAPI.GroupsDeleteGroup`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**groupName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiGroupsDeleteGroupRequest struct via the builder pattern


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


## GroupsDescribeGroup

> GroupsDescribeGroupResponse GroupsDescribeGroup(ctx, groupName).DomainType(domainType).DomainId(domainId).Execute()

Get group



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
	groupName := "groupName_example" // string | 
	domainType := "domainType_example" // string |  (optional) (default to "DOMAIN_TYPE_UNSPECIFIED")
	domainId := "domainId_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.GroupsAPI.GroupsDescribeGroup(context.Background(), groupName).DomainType(domainType).DomainId(domainId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `GroupsAPI.GroupsDescribeGroup``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GroupsDescribeGroup`: GroupsDescribeGroupResponse
	fmt.Fprintf(os.Stdout, "Response from `GroupsAPI.GroupsDescribeGroup`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**groupName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiGroupsDescribeGroupRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **domainType** | **string** |  | [default to &quot;DOMAIN_TYPE_UNSPECIFIED&quot;]
 **domainId** | **string** |  | 

### Return type

[**GroupsDescribeGroupResponse**](GroupsDescribeGroupResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GroupsListGroupUsers

> GroupsListGroupUsersResponse GroupsListGroupUsers(ctx, groupName).DomainType(domainType).DomainId(domainId).Execute()

Get group users



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
	groupName := "groupName_example" // string | 
	domainType := "domainType_example" // string |  (optional) (default to "DOMAIN_TYPE_UNSPECIFIED")
	domainId := "domainId_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.GroupsAPI.GroupsListGroupUsers(context.Background(), groupName).DomainType(domainType).DomainId(domainId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `GroupsAPI.GroupsListGroupUsers``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GroupsListGroupUsers`: GroupsListGroupUsersResponse
	fmt.Fprintf(os.Stdout, "Response from `GroupsAPI.GroupsListGroupUsers`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**groupName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiGroupsListGroupUsersRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **domainType** | **string** |  | [default to &quot;DOMAIN_TYPE_UNSPECIFIED&quot;]
 **domainId** | **string** |  | 

### Return type

[**GroupsListGroupUsersResponse**](GroupsListGroupUsersResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GroupsListGroups

> GroupsListGroupsResponse GroupsListGroups(ctx).DomainType(domainType).DomainId(domainId).Execute()

List groups



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
	resp, r, err := apiClient.GroupsAPI.GroupsListGroups(context.Background()).DomainType(domainType).DomainId(domainId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `GroupsAPI.GroupsListGroups``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GroupsListGroups`: GroupsListGroupsResponse
	fmt.Fprintf(os.Stdout, "Response from `GroupsAPI.GroupsListGroups`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiGroupsListGroupsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **domainType** | **string** |  | [default to &quot;DOMAIN_TYPE_UNSPECIFIED&quot;]
 **domainId** | **string** |  | 

### Return type

[**GroupsListGroupsResponse**](GroupsListGroupsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GroupsRemoveUserFromGroup

> map[string]interface{} GroupsRemoveUserFromGroup(ctx, groupName).Body(body).Execute()

Remove user from group



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
	groupName := "groupName_example" // string | 
	body := *openapiclient.NewGroupsRemoveUserFromGroupBody() // GroupsRemoveUserFromGroupBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.GroupsAPI.GroupsRemoveUserFromGroup(context.Background(), groupName).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `GroupsAPI.GroupsRemoveUserFromGroup``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GroupsRemoveUserFromGroup`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `GroupsAPI.GroupsRemoveUserFromGroup`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**groupName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiGroupsRemoveUserFromGroupRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**GroupsRemoveUserFromGroupBody**](GroupsRemoveUserFromGroupBody.md) |  | 

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


## GroupsUpdateGroup

> GroupsUpdateGroupResponse GroupsUpdateGroup(ctx, groupName).Body(body).Execute()

Update group



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
	groupName := "groupName_example" // string | 
	body := *openapiclient.NewGroupsUpdateGroupBody() // GroupsUpdateGroupBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.GroupsAPI.GroupsUpdateGroup(context.Background(), groupName).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `GroupsAPI.GroupsUpdateGroup``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `GroupsUpdateGroup`: GroupsUpdateGroupResponse
	fmt.Fprintf(os.Stdout, "Response from `GroupsAPI.GroupsUpdateGroup`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**groupName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiGroupsUpdateGroupRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**GroupsUpdateGroupBody**](GroupsUpdateGroupBody.md) |  | 

### Return type

[**GroupsUpdateGroupResponse**](GroupsUpdateGroupResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

