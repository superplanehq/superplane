# \OrganizationAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**OrganizationsAcceptInviteLink**](OrganizationAPI.md#OrganizationsAcceptInviteLink) | **Post** /api/v1/invite-links/{token}/accept | Accept an invite link
[**OrganizationsCreateIntegration**](OrganizationAPI.md#OrganizationsCreateIntegration) | **Post** /api/v1/organizations/{id}/integrations | Create organization integration
[**OrganizationsCreateInvitation**](OrganizationAPI.md#OrganizationsCreateInvitation) | **Post** /api/v1/organizations/{id}/invitations | Create an organization invitation
[**OrganizationsDeleteIntegration**](OrganizationAPI.md#OrganizationsDeleteIntegration) | **Delete** /api/v1/organizations/{id}/integrations/{integrationId} | Delete organization integration
[**OrganizationsDeleteOrganization**](OrganizationAPI.md#OrganizationsDeleteOrganization) | **Delete** /api/v1/organizations/{id} | Delete an organization
[**OrganizationsDescribeIntegration**](OrganizationAPI.md#OrganizationsDescribeIntegration) | **Get** /api/v1/organizations/{id}/integrations/{integrationId} | Describe an integration in an organization
[**OrganizationsDescribeOrganization**](OrganizationAPI.md#OrganizationsDescribeOrganization) | **Get** /api/v1/organizations/{id} | Get organization details
[**OrganizationsGetInviteLink**](OrganizationAPI.md#OrganizationsGetInviteLink) | **Get** /api/v1/organizations/{id}/invite-link | Get an organization invite link
[**OrganizationsListIntegrationResources**](OrganizationAPI.md#OrganizationsListIntegrationResources) | **Get** /api/v1/organizations/{id}/integrations/{integrationId}/resources | List integration resources
[**OrganizationsListIntegrations**](OrganizationAPI.md#OrganizationsListIntegrations) | **Get** /api/v1/organizations/{id}/integrations | List integrations in an organization
[**OrganizationsListInvitations**](OrganizationAPI.md#OrganizationsListInvitations) | **Get** /api/v1/organizations/{id}/invitations | List organization invitations
[**OrganizationsRemoveInvitation**](OrganizationAPI.md#OrganizationsRemoveInvitation) | **Delete** /api/v1/organizations/{id}/invitations/{invitationId} | Remove an organization invitation
[**OrganizationsRemoveUser**](OrganizationAPI.md#OrganizationsRemoveUser) | **Delete** /api/v1/organizations/{id}/users/{userId} | Remove a user from an organization
[**OrganizationsResetInviteLink**](OrganizationAPI.md#OrganizationsResetInviteLink) | **Post** /api/v1/organizations/{id}/invite-link/reset | Reset an organization invite link
[**OrganizationsUpdateIntegration**](OrganizationAPI.md#OrganizationsUpdateIntegration) | **Patch** /api/v1/organizations/{id}/integrations/{integrationId} | Update integration
[**OrganizationsUpdateInviteLink**](OrganizationAPI.md#OrganizationsUpdateInviteLink) | **Patch** /api/v1/organizations/{id}/invite-link | Update an organization invite link
[**OrganizationsUpdateOrganization**](OrganizationAPI.md#OrganizationsUpdateOrganization) | **Patch** /api/v1/organizations/{id} | Update an organization



## OrganizationsAcceptInviteLink

> map[string]interface{} OrganizationsAcceptInviteLink(ctx, token).Id(id).OrganizationId(organizationId).Enabled(enabled).CreatedAt(createdAt).UpdatedAt(updatedAt).Execute()

Accept an invite link



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
	token := "token_example" // string | 
	id := "id_example" // string |  (optional)
	organizationId := "organizationId_example" // string |  (optional)
	enabled := true // bool |  (optional)
	createdAt := time.Now() // time.Time |  (optional)
	updatedAt := time.Now() // time.Time |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.OrganizationAPI.OrganizationsAcceptInviteLink(context.Background(), token).Id(id).OrganizationId(organizationId).Enabled(enabled).CreatedAt(createdAt).UpdatedAt(updatedAt).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsAcceptInviteLink``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsAcceptInviteLink`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsAcceptInviteLink`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**token** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsAcceptInviteLinkRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **id** | **string** |  | 
 **organizationId** | **string** |  | 
 **enabled** | **bool** |  | 
 **createdAt** | **time.Time** |  | 
 **updatedAt** | **time.Time** |  | 

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


## OrganizationsCreateIntegration

> OrganizationsCreateIntegrationResponse OrganizationsCreateIntegration(ctx, id).Body(body).Execute()

Create organization integration



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
	body := *openapiclient.NewOrganizationsCreateIntegrationBody() // OrganizationsCreateIntegrationBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.OrganizationAPI.OrganizationsCreateIntegration(context.Background(), id).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsCreateIntegration``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsCreateIntegration`: OrganizationsCreateIntegrationResponse
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsCreateIntegration`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsCreateIntegrationRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**OrganizationsCreateIntegrationBody**](OrganizationsCreateIntegrationBody.md) |  | 

### Return type

[**OrganizationsCreateIntegrationResponse**](OrganizationsCreateIntegrationResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrganizationsCreateInvitation

> OrganizationsCreateInvitationResponse OrganizationsCreateInvitation(ctx, id).Body(body).Execute()

Create an organization invitation



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
	body := *openapiclient.NewOrganizationsCreateInvitationBody() // OrganizationsCreateInvitationBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.OrganizationAPI.OrganizationsCreateInvitation(context.Background(), id).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsCreateInvitation``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsCreateInvitation`: OrganizationsCreateInvitationResponse
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsCreateInvitation`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsCreateInvitationRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**OrganizationsCreateInvitationBody**](OrganizationsCreateInvitationBody.md) |  | 

### Return type

[**OrganizationsCreateInvitationResponse**](OrganizationsCreateInvitationResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrganizationsDeleteIntegration

> map[string]interface{} OrganizationsDeleteIntegration(ctx, id, integrationId).Execute()

Delete organization integration



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
	integrationId := "integrationId_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.OrganizationAPI.OrganizationsDeleteIntegration(context.Background(), id, integrationId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsDeleteIntegration``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsDeleteIntegration`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsDeleteIntegration`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 
**integrationId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsDeleteIntegrationRequest struct via the builder pattern


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


## OrganizationsDeleteOrganization

> map[string]interface{} OrganizationsDeleteOrganization(ctx, id).Execute()

Delete an organization



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
	resp, r, err := apiClient.OrganizationAPI.OrganizationsDeleteOrganization(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsDeleteOrganization``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsDeleteOrganization`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsDeleteOrganization`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsDeleteOrganizationRequest struct via the builder pattern


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


## OrganizationsDescribeIntegration

> OrganizationsDescribeIntegrationResponse OrganizationsDescribeIntegration(ctx, id, integrationId).Execute()

Describe an integration in an organization



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
	integrationId := "integrationId_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.OrganizationAPI.OrganizationsDescribeIntegration(context.Background(), id, integrationId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsDescribeIntegration``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsDescribeIntegration`: OrganizationsDescribeIntegrationResponse
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsDescribeIntegration`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 
**integrationId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsDescribeIntegrationRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**OrganizationsDescribeIntegrationResponse**](OrganizationsDescribeIntegrationResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrganizationsDescribeOrganization

> OrganizationsDescribeOrganizationResponse OrganizationsDescribeOrganization(ctx, id).Execute()

Get organization details



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
	resp, r, err := apiClient.OrganizationAPI.OrganizationsDescribeOrganization(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsDescribeOrganization``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsDescribeOrganization`: OrganizationsDescribeOrganizationResponse
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsDescribeOrganization`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsDescribeOrganizationRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**OrganizationsDescribeOrganizationResponse**](OrganizationsDescribeOrganizationResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrganizationsGetInviteLink

> OrganizationsGetInviteLinkResponse OrganizationsGetInviteLink(ctx, id).Execute()

Get an organization invite link



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
	resp, r, err := apiClient.OrganizationAPI.OrganizationsGetInviteLink(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsGetInviteLink``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsGetInviteLink`: OrganizationsGetInviteLinkResponse
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsGetInviteLink`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsGetInviteLinkRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**OrganizationsGetInviteLinkResponse**](OrganizationsGetInviteLinkResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrganizationsListIntegrationResources

> OrganizationsListIntegrationResourcesResponse OrganizationsListIntegrationResources(ctx, id, integrationId).Type_(type_).Execute()

List integration resources



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
	integrationId := "integrationId_example" // string | 
	type_ := "type__example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.OrganizationAPI.OrganizationsListIntegrationResources(context.Background(), id, integrationId).Type_(type_).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsListIntegrationResources``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsListIntegrationResources`: OrganizationsListIntegrationResourcesResponse
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsListIntegrationResources`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 
**integrationId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsListIntegrationResourcesRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **type_** | **string** |  | 

### Return type

[**OrganizationsListIntegrationResourcesResponse**](OrganizationsListIntegrationResourcesResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrganizationsListIntegrations

> SuperplaneOrganizationsListIntegrationsResponse OrganizationsListIntegrations(ctx, id).Execute()

List integrations in an organization



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
	resp, r, err := apiClient.OrganizationAPI.OrganizationsListIntegrations(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsListIntegrations``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsListIntegrations`: SuperplaneOrganizationsListIntegrationsResponse
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsListIntegrations`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsListIntegrationsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**SuperplaneOrganizationsListIntegrationsResponse**](SuperplaneOrganizationsListIntegrationsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrganizationsListInvitations

> OrganizationsListInvitationsResponse OrganizationsListInvitations(ctx, id).Execute()

List organization invitations



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
	resp, r, err := apiClient.OrganizationAPI.OrganizationsListInvitations(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsListInvitations``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsListInvitations`: OrganizationsListInvitationsResponse
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsListInvitations`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsListInvitationsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**OrganizationsListInvitationsResponse**](OrganizationsListInvitationsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrganizationsRemoveInvitation

> map[string]interface{} OrganizationsRemoveInvitation(ctx, id, invitationId).Execute()

Remove an organization invitation



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
	invitationId := "invitationId_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.OrganizationAPI.OrganizationsRemoveInvitation(context.Background(), id, invitationId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsRemoveInvitation``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsRemoveInvitation`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsRemoveInvitation`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 
**invitationId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsRemoveInvitationRequest struct via the builder pattern


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


## OrganizationsRemoveUser

> map[string]interface{} OrganizationsRemoveUser(ctx, id, userId).Execute()

Remove a user from an organization



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
	userId := "userId_example" // string | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.OrganizationAPI.OrganizationsRemoveUser(context.Background(), id, userId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsRemoveUser``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsRemoveUser`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsRemoveUser`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 
**userId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsRemoveUserRequest struct via the builder pattern


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


## OrganizationsResetInviteLink

> OrganizationsResetInviteLinkResponse OrganizationsResetInviteLink(ctx, id).Execute()

Reset an organization invite link



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
	resp, r, err := apiClient.OrganizationAPI.OrganizationsResetInviteLink(context.Background(), id).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsResetInviteLink``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsResetInviteLink`: OrganizationsResetInviteLinkResponse
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsResetInviteLink`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsResetInviteLinkRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**OrganizationsResetInviteLinkResponse**](OrganizationsResetInviteLinkResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrganizationsUpdateIntegration

> OrganizationsUpdateIntegrationResponse OrganizationsUpdateIntegration(ctx, id, integrationId).Body(body).Execute()

Update integration



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
	integrationId := "integrationId_example" // string | 
	body := *openapiclient.NewOrganizationsUpdateIntegrationBody() // OrganizationsUpdateIntegrationBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.OrganizationAPI.OrganizationsUpdateIntegration(context.Background(), id, integrationId).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsUpdateIntegration``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsUpdateIntegration`: OrganizationsUpdateIntegrationResponse
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsUpdateIntegration`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 
**integrationId** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsUpdateIntegrationRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | [**OrganizationsUpdateIntegrationBody**](OrganizationsUpdateIntegrationBody.md) |  | 

### Return type

[**OrganizationsUpdateIntegrationResponse**](OrganizationsUpdateIntegrationResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrganizationsUpdateInviteLink

> OrganizationsUpdateInviteLinkResponse OrganizationsUpdateInviteLink(ctx, id).Body(body).Execute()

Update an organization invite link



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
	body := *openapiclient.NewOrganizationsUpdateInviteLinkBody() // OrganizationsUpdateInviteLinkBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.OrganizationAPI.OrganizationsUpdateInviteLink(context.Background(), id).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsUpdateInviteLink``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsUpdateInviteLink`: OrganizationsUpdateInviteLinkResponse
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsUpdateInviteLink`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsUpdateInviteLinkRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**OrganizationsUpdateInviteLinkBody**](OrganizationsUpdateInviteLinkBody.md) |  | 

### Return type

[**OrganizationsUpdateInviteLinkResponse**](OrganizationsUpdateInviteLinkResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## OrganizationsUpdateOrganization

> OrganizationsUpdateOrganizationResponse OrganizationsUpdateOrganization(ctx, id).Body(body).Execute()

Update an organization



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
	body := *openapiclient.NewOrganizationsUpdateOrganizationBody() // OrganizationsUpdateOrganizationBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.OrganizationAPI.OrganizationsUpdateOrganization(context.Background(), id).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `OrganizationAPI.OrganizationsUpdateOrganization``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `OrganizationsUpdateOrganization`: OrganizationsUpdateOrganizationResponse
	fmt.Fprintf(os.Stdout, "Response from `OrganizationAPI.OrganizationsUpdateOrganization`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiOrganizationsUpdateOrganizationRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**OrganizationsUpdateOrganizationBody**](OrganizationsUpdateOrganizationBody.md) |  | 

### Return type

[**OrganizationsUpdateOrganizationResponse**](OrganizationsUpdateOrganizationResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

