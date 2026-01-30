# SecretsSecretSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Provider** | Pointer to [**SecretProvider**](SecretProvider.md) |  | [optional] [default to SECRETPROVIDER_PROVIDER_UNKNOWN]
**Local** | Pointer to [**SecretLocal**](SecretLocal.md) |  | [optional] 

## Methods

### NewSecretsSecretSpec

`func NewSecretsSecretSpec() *SecretsSecretSpec`

NewSecretsSecretSpec instantiates a new SecretsSecretSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSecretsSecretSpecWithDefaults

`func NewSecretsSecretSpecWithDefaults() *SecretsSecretSpec`

NewSecretsSecretSpecWithDefaults instantiates a new SecretsSecretSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetProvider

`func (o *SecretsSecretSpec) GetProvider() SecretProvider`

GetProvider returns the Provider field if non-nil, zero value otherwise.

### GetProviderOk

`func (o *SecretsSecretSpec) GetProviderOk() (*SecretProvider, bool)`

GetProviderOk returns a tuple with the Provider field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetProvider

`func (o *SecretsSecretSpec) SetProvider(v SecretProvider)`

SetProvider sets Provider field to given value.

### HasProvider

`func (o *SecretsSecretSpec) HasProvider() bool`

HasProvider returns a boolean if a field has been set.

### GetLocal

`func (o *SecretsSecretSpec) GetLocal() SecretLocal`

GetLocal returns the Local field if non-nil, zero value otherwise.

### GetLocalOk

`func (o *SecretsSecretSpec) GetLocalOk() (*SecretLocal, bool)`

GetLocalOk returns a tuple with the Local field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLocal

`func (o *SecretsSecretSpec) SetLocal(v SecretLocal)`

SetLocal sets Local field to given value.

### HasLocal

`func (o *SecretsSecretSpec) HasLocal() bool`

HasLocal returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


