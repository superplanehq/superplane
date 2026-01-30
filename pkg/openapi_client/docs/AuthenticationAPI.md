# \AuthenticationAPI

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AuthGetConfig**](AuthenticationAPI.md#AuthGetConfig) | **Get** /auth/config | Get authentication configuration
[**AuthLogin**](AuthenticationAPI.md#AuthLogin) | **Post** /login | Login with password
[**AuthSignup**](AuthenticationAPI.md#AuthSignup) | **Post** /signup | Sign up with password



## AuthGetConfig

> AuthConfig AuthGetConfig(ctx).Execute()

Get authentication configuration



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
	resp, r, err := apiClient.AuthenticationAPI.AuthGetConfig(context.Background()).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthenticationAPI.AuthGetConfig``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
	// response from `AuthGetConfig`: AuthConfig
	fmt.Fprintf(os.Stdout, "Response from `AuthenticationAPI.AuthGetConfig`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiAuthGetConfigRequest struct via the builder pattern


### Return type

[**AuthConfig**](AuthConfig.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthLogin

> AuthLogin(ctx).Email(email).Password(password).Redirect(redirect).Execute()

Login with password



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
	email := "email_example" // string | User email address
	password := "password_example" // string | User password
	redirect := "redirect_example" // string | Redirect URL after successful login (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.AuthenticationAPI.AuthLogin(context.Background()).Email(email).Password(password).Redirect(redirect).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthenticationAPI.AuthLogin``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAuthLoginRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **email** | **string** | User email address | 
 **password** | **string** | User password | 
 **redirect** | **string** | Redirect URL after successful login | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/x-www-form-urlencoded
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## AuthSignup

> AuthSignup(ctx).Name(name).Email(email).Password(password).Redirect(redirect).InviteToken(inviteToken).Execute()

Sign up with password



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
	name := "name_example" // string | Full name
	email := "email_example" // string | Email address
	password := "password_example" // string | Password
	redirect := "redirect_example" // string | Redirect URL after successful signup (optional)
	inviteToken := "inviteToken_example" // string | Optional invitation token (optional)

	configuration := openapiclient.NewConfiguration()
	apiClient := openapiclient.NewAPIClient(configuration)
	r, err := apiClient.AuthenticationAPI.AuthSignup(context.Background()).Name(name).Email(email).Password(password).Redirect(redirect).InviteToken(inviteToken).Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error when calling `AuthenticationAPI.AuthSignup``: %v\n", err)
		fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
	}
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiAuthSignupRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **name** | **string** | Full name | 
 **email** | **string** | Email address | 
 **password** | **string** | Password | 
 **redirect** | **string** | Redirect URL after successful signup | 
 **inviteToken** | **string** | Optional invitation token | 

### Return type

 (empty response body)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: application/x-www-form-urlencoded
- **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

