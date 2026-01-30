# \SetupAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**SetupSetupOwner**](SetupAPI.md#SetupSetupOwner) | **Post** /api/v1/setup-owner | Setup owner account



## SetupSetupOwner

> SetupOwnerResponse SetupSetupOwner(ctx).Body(body).Execute()

Setup owner account



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
	body := *openapiclient.NewSetupOwnerRequest("Email_example", "FirstName_example", "LastName_example", "Password_example") // SetupOwnerRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SetupAPI.SetupSetupOwner(context.Background()).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SetupAPI.SetupSetupOwner``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `SetupSetupOwner`: SetupOwnerResponse
	fmt.Fprintf(os.Stdout, "Response from `SetupAPI.SetupSetupOwner`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiSetupSetupOwnerRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**SetupOwnerRequest**](SetupOwnerRequest.md) |  | 

### Return type

[**SetupOwnerResponse**](SetupOwnerResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

