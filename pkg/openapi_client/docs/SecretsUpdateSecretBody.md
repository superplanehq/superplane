# SecretsUpdateSecretBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Secret** | Pointer to [**SecretsSecret**](SecretsSecret.md) |  | [optional] 
**DomainType** | Pointer to [**AuthorizationDomainType**](AuthorizationDomainType.md) |  | [optional] [default to AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_UNSPECIFIED]
**DomainId** | Pointer to **string** |  | [optional] 

## Methods

### NewSecretsUpdateSecretBody

`func NewSecretsUpdateSecretBody() *SecretsUpdateSecretBody`

NewSecretsUpdateSecretBody instantiates a new SecretsUpdateSecretBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSecretsUpdateSecretBodyWithDefaults

`func NewSecretsUpdateSecretBodyWithDefaults() *SecretsUpdateSecretBody`

NewSecretsUpdateSecretBodyWithDefaults instantiates a new SecretsUpdateSecretBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSecret

`func (o *SecretsUpdateSecretBody) GetSecret() SecretsSecret`

GetSecret returns the Secret field if non-nil, zero value otherwise.

### GetSecretOk

`func (o *SecretsUpdateSecretBody) GetSecretOk() (*SecretsSecret, bool)`

GetSecretOk returns a tuple with the Secret field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecret

`func (o *SecretsUpdateSecretBody) SetSecret(v SecretsSecret)`

SetSecret sets Secret field to given value.

### HasSecret

`func (o *SecretsUpdateSecretBody) HasSecret() bool`

HasSecret returns a boolean if a field has been set.

### GetDomainType

`func (o *SecretsUpdateSecretBody) GetDomainType() AuthorizationDomainType`

GetDomainType returns the DomainType field if non-nil, zero value otherwise.

### GetDomainTypeOk

`func (o *SecretsUpdateSecretBody) GetDomainTypeOk() (*AuthorizationDomainType, bool)`

GetDomainTypeOk returns a tuple with the DomainType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainType

`func (o *SecretsUpdateSecretBody) SetDomainType(v AuthorizationDomainType)`

SetDomainType sets DomainType field to given value.

### HasDomainType

`func (o *SecretsUpdateSecretBody) HasDomainType() bool`

HasDomainType returns a boolean if a field has been set.

### GetDomainId

`func (o *SecretsUpdateSecretBody) GetDomainId() string`

GetDomainId returns the DomainId field if non-nil, zero value otherwise.

### GetDomainIdOk

`func (o *SecretsUpdateSecretBody) GetDomainIdOk() (*string, bool)`

GetDomainIdOk returns a tuple with the DomainId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainId

`func (o *SecretsUpdateSecretBody) SetDomainId(v string)`

SetDomainId sets DomainId field to given value.

### HasDomainId

`func (o *SecretsUpdateSecretBody) HasDomainId() bool`

HasDomainId returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


