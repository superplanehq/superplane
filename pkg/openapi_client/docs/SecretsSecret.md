# SecretsSecret

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Metadata** | Pointer to [**SecretsSecretMetadata**](SecretsSecretMetadata.md) |  | [optional] 
**Spec** | Pointer to [**SecretsSecretSpec**](SecretsSecretSpec.md) |  | [optional] 

## Methods

### NewSecretsSecret

`func NewSecretsSecret() *SecretsSecret`

NewSecretsSecret instantiates a new SecretsSecret object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSecretsSecretWithDefaults

`func NewSecretsSecretWithDefaults() *SecretsSecret`

NewSecretsSecretWithDefaults instantiates a new SecretsSecret object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMetadata

`func (o *SecretsSecret) GetMetadata() SecretsSecretMetadata`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *SecretsSecret) GetMetadataOk() (*SecretsSecretMetadata, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *SecretsSecret) SetMetadata(v SecretsSecretMetadata)`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *SecretsSecret) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetSpec

`func (o *SecretsSecret) GetSpec() SecretsSecretSpec`

GetSpec returns the Spec field if non-nil, zero value otherwise.

### GetSpecOk

`func (o *SecretsSecret) GetSpecOk() (*SecretsSecretSpec, bool)`

GetSpecOk returns a tuple with the Spec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSpec

`func (o *SecretsSecret) SetSpec(v SecretsSecretSpec)`

SetSpec sets Spec field to given value.

### HasSpec

`func (o *SecretsSecret) HasSpec() bool`

HasSpec returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


