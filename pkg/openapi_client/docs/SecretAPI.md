# \SecretAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**SecretsCreateSecret**](SecretAPI.md#SecretsCreateSecret) | **Post** /api/v1/secrets | Create a new secret
[**SecretsDeleteSecret**](SecretAPI.md#SecretsDeleteSecret) | **Delete** /api/v1/secrets/{idOrName} | Deletes a secret
[**SecretsDescribeSecret**](SecretAPI.md#SecretsDescribeSecret) | **Get** /api/v1/secrets/{idOrName} | Get secret details
[**SecretsListSecrets**](SecretAPI.md#SecretsListSecrets) | **Get** /api/v1/secrets | List secrets
[**SecretsUpdateSecret**](SecretAPI.md#SecretsUpdateSecret) | **Patch** /api/v1/secrets/{idOrName} | Updates a secret



## SecretsCreateSecret

> SecretsCreateSecretResponse SecretsCreateSecret(ctx).Body(body).Execute()

Create a new secret



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
	body := *openapiclient.NewSecretsCreateSecretRequest() // SecretsCreateSecretRequest | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SecretAPI.SecretsCreateSecret(context.Background()).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SecretAPI.SecretsCreateSecret``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `SecretsCreateSecret`: SecretsCreateSecretResponse
	fmt.Fprintf(os.Stdout, "Response from `SecretAPI.SecretsCreateSecret`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiSecretsCreateSecretRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | [**SecretsCreateSecretRequest**](SecretsCreateSecretRequest.md) |  | 

### Return type

[**SecretsCreateSecretResponse**](SecretsCreateSecretResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SecretsDeleteSecret

> map[string]interface{} SecretsDeleteSecret(ctx, idOrName).DomainType(domainType).DomainId(domainId).Execute()

Deletes a secret



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
	idOrName := "idOrName_example" // string | 
	domainType := "domainType_example" // string |  (optional) (default to "DOMAIN_TYPE_UNSPECIFIED")
	domainId := "domainId_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SecretAPI.SecretsDeleteSecret(context.Background(), idOrName).DomainType(domainType).DomainId(domainId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SecretAPI.SecretsDeleteSecret``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `SecretsDeleteSecret`: map[string]interface{}
	fmt.Fprintf(os.Stdout, "Response from `SecretAPI.SecretsDeleteSecret`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**idOrName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiSecretsDeleteSecretRequest struct via the builder pattern


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


## SecretsDescribeSecret

> SecretsDescribeSecretResponse SecretsDescribeSecret(ctx, idOrName).DomainType(domainType).DomainId(domainId).Execute()

Get secret details



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
	idOrName := "idOrName_example" // string | 
	domainType := "domainType_example" // string |  (optional) (default to "DOMAIN_TYPE_UNSPECIFIED")
	domainId := "domainId_example" // string |  (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SecretAPI.SecretsDescribeSecret(context.Background(), idOrName).DomainType(domainType).DomainId(domainId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SecretAPI.SecretsDescribeSecret``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `SecretsDescribeSecret`: SecretsDescribeSecretResponse
	fmt.Fprintf(os.Stdout, "Response from `SecretAPI.SecretsDescribeSecret`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**idOrName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiSecretsDescribeSecretRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **domainType** | **string** |  | [default to &quot;DOMAIN_TYPE_UNSPECIFIED&quot;]
 **domainId** | **string** |  | 

### Return type

[**SecretsDescribeSecretResponse**](SecretsDescribeSecretResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SecretsListSecrets

> SecretsListSecretsResponse SecretsListSecrets(ctx).DomainType(domainType).DomainId(domainId).Execute()

List secrets



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
	resp, r, err := apiClient.SecretAPI.SecretsListSecrets(context.Background()).DomainType(domainType).DomainId(domainId).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SecretAPI.SecretsListSecrets``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `SecretsListSecrets`: SecretsListSecretsResponse
	fmt.Fprintf(os.Stdout, "Response from `SecretAPI.SecretsListSecrets`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiSecretsListSecretsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **domainType** | **string** |  | [default to &quot;DOMAIN_TYPE_UNSPECIFIED&quot;]
 **domainId** | **string** |  | 

### Return type

[**SecretsListSecretsResponse**](SecretsListSecretsResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SecretsUpdateSecret

> SecretsUpdateSecretResponse SecretsUpdateSecret(ctx, idOrName).Body(body).Execute()

Updates a secret



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
	idOrName := "idOrName_example" // string | 
	body := *openapiclient.NewSecretsUpdateSecretBody() // SecretsUpdateSecretBody | 

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	resp, r, err := apiClient.SecretAPI.SecretsUpdateSecret(context.Background(), idOrName).Body(body).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `SecretAPI.SecretsUpdateSecret``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `SecretsUpdateSecret`: SecretsUpdateSecretResponse
	fmt.Fprintf(os.Stdout, "Response from `SecretAPI.SecretsUpdateSecret`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**idOrName** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiSecretsUpdateSecretRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | [**SecretsUpdateSecretBody**](SecretsUpdateSecretBody.md) |  | 

### Return type

[**SecretsUpdateSecretResponse**](SecretsUpdateSecretResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

