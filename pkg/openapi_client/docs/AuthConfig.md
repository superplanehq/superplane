# AuthConfig

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Providers** | Pointer to **[]string** | Available OAuth providers (e.g., &#39;github&#39;, &#39;google&#39;) | [optional] 
**PasswordLoginEnabled** | Pointer to **bool** | Whether password-based login is enabled | [optional] 
**SignupEnabled** | Pointer to **bool** | Whether new user signup is enabled | [optional] 

## Methods

### NewAuthConfig

`func NewAuthConfig() *AuthConfig`

NewAuthConfig instantiates a new AuthConfig object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAuthConfigWithDefaults

`func NewAuthConfigWithDefaults() *AuthConfig`

NewAuthConfigWithDefaults instantiates a new AuthConfig object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetProviders

`func (o *AuthConfig) GetProviders() []string`

GetProviders returns the Providers field if non-nil, zero value otherwise.

### GetProvidersOk

`func (o *AuthConfig) GetProvidersOk() (*[]string, bool)`

GetProvidersOk returns a tuple with the Providers field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProviders

`func (o *AuthConfig) SetProviders(v []string)`

SetProviders sets Providers field to given value.

### HasProviders

`func (o *AuthConfig) HasProviders() bool`

HasProviders returns a boolean if a field has been set.

### GetPasswordLoginEnabled

`func (o *AuthConfig) GetPasswordLoginEnabled() bool`

GetPasswordLoginEnabled returns the PasswordLoginEnabled field if non-nil, zero value otherwise.

### GetPasswordLoginEnabledOk

`func (o *AuthConfig) GetPasswordLoginEnabledOk() (*bool, bool)`

GetPasswordLoginEnabledOk returns a tuple with the PasswordLoginEnabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPasswordLoginEnabled

`func (o *AuthConfig) SetPasswordLoginEnabled(v bool)`

SetPasswordLoginEnabled sets PasswordLoginEnabled field to given value.

### HasPasswordLoginEnabled

`func (o *AuthConfig) HasPasswordLoginEnabled() bool`

HasPasswordLoginEnabled returns a boolean if a field has been set.

### GetSignupEnabled

`func (o *AuthConfig) GetSignupEnabled() bool`

GetSignupEnabled returns the SignupEnabled field if non-nil, zero value otherwise.

### GetSignupEnabledOk

`func (o *AuthConfig) GetSignupEnabledOk() (*bool, bool)`

GetSignupEnabledOk returns a tuple with the SignupEnabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSignupEnabled

`func (o *AuthConfig) SetSignupEnabled(v bool)`

SetSignupEnabled sets SignupEnabled field to given value.

### HasSignupEnabled

`func (o *AuthConfig) HasSignupEnabled() bool`

HasSignupEnabled returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


