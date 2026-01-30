# \MeAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**MeMe**](MeAPI.md#MeMe) | **Get** /api/v1/me | Get current user
[**MeRegenerateToken**](MeAPI.md#MeRegenerateToken) | **Post** /api/v1/me/token | Regenerate API token



## MeMe

> SuperplaneMeUser MeMe(ctx).Execute()

Get current user



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
	resp, r, err := apiClient.MeAPI.MeMe(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `MeAPI.MeMe``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `MeMe`: SuperplaneMeUser
	fmt.Fprintf(os.Stdout, "Response from `MeAPI.MeMe`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiMeMeRequest struct via the builder pattern


### Return type

[**SuperplaneMeUser**](SuperplaneMeUser.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## MeRegenerateToken

> MeRegenerateTokenResponse MeRegenerateToken(ctx).Execute()

Regenerate API token



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
	resp, r, err := apiClient.MeAPI.MeRegenerateToken(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `MeAPI.MeRegenerateToken``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `MeRegenerateToken`: MeRegenerateTokenResponse
	fmt.Fprintf(os.Stdout, "Response from `MeAPI.MeRegenerateToken`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiMeRegenerateTokenRequest struct via the builder pattern


### Return type

[**MeRegenerateTokenResponse**](MeRegenerateTokenResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

