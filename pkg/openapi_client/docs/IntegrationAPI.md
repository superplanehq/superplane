# \IntegrationAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**IntegrationsListIntegrations**](IntegrationAPI.md#IntegrationsListIntegrations) | **Get** /api/v1/integrations | List available integrations



## IntegrationsListIntegrations

> SuperplaneIntegrationsListIntegrationsResponse IntegrationsListIntegrations(ctx).Execute()

List available integrations



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
	resp, r, err := apiClient.IntegrationAPI.IntegrationsListIntegrations(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `IntegrationAPI.IntegrationsListIntegrations``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `IntegrationsListIntegrations`: SuperplaneIntegrationsListIntegrationsResponse
	fmt.Fprintf(os.Stdout, "Response from `IntegrationAPI.IntegrationsListIntegrations`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiIntegrationsListIntegrationsRequest struct via the builder pattern


### Return type

[**SuperplaneIntegrationsListIntegrationsResponse**](SuperplaneIntegrationsListIntegrationsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

