# SecretsSecretMetadata

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Name** | Pointer to **string** |  | [optional] 
**DomainType** | Pointer to [**AuthorizationDomainType**](AuthorizationDomainType.md) |  | [optional] [default to AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_UNSPECIFIED]
**DomainId** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 

## Methods

### NewSecretsSecretMetadata

`func NewSecretsSecretMetadata() *SecretsSecretMetadata`

NewSecretsSecretMetadata instantiates a new SecretsSecretMetadata object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSecretsSecretMetadataWithDefaults

`func NewSecretsSecretMetadataWithDefaults() *SecretsSecretMetadata`

NewSecretsSecretMetadataWithDefaults instantiates a new SecretsSecretMetadata object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *SecretsSecretMetadata) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *SecretsSecretMetadata) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *SecretsSecretMetadata) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *SecretsSecretMetadata) HasId() bool`

HasId returns a boolean if a field has been set.

### GetName

`func (o *SecretsSecretMetadata) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *SecretsSecretMetadata) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *SecretsSecretMetadata) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *SecretsSecretMetadata) HasName() bool`

HasName returns a boolean if a field has been set.

### GetDomainType

`func (o *SecretsSecretMetadata) GetDomainType() AuthorizationDomainType`

GetDomainType returns the DomainType field if non-nil, zero value otherwise.

### GetDomainTypeOk

`func (o *SecretsSecretMetadata) GetDomainTypeOk() (*AuthorizationDomainType, bool)`

GetDomainTypeOk returns a tuple with the DomainType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainType

`func (o *SecretsSecretMetadata) SetDomainType(v AuthorizationDomainType)`

SetDomainType sets DomainType field to given value.

### HasDomainType

`func (o *SecretsSecretMetadata) HasDomainType() bool`

HasDomainType returns a boolean if a field has been set.

### GetDomainId

`func (o *SecretsSecretMetadata) GetDomainId() string`

GetDomainId returns the DomainId field if non-nil, zero value otherwise.

### GetDomainIdOk

`func (o *SecretsSecretMetadata) GetDomainIdOk() (*string, bool)`

GetDomainIdOk returns a tuple with the DomainId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainId

`func (o *SecretsSecretMetadata) SetDomainId(v string)`

SetDomainId sets DomainId field to given value.

### HasDomainId

`func (o *SecretsSecretMetadata) HasDomainId() bool`

HasDomainId returns a boolean if a field has been set.

### GetCreatedAt

`func (o *SecretsSecretMetadata) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *SecretsSecretMetadata) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *SecretsSecretMetadata) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *SecretsSecretMetadata) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


