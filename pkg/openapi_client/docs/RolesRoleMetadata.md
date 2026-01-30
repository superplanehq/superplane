# RolesRoleMetadata

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** |  | [optional] 
**DomainType** | Pointer to [**AuthorizationDomainType**](AuthorizationDomainType.md) |  | [optional] [default to AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_UNSPECIFIED]
**DomainId** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 

## Methods

### NewRolesRoleMetadata

`func NewRolesRoleMetadata() *RolesRoleMetadata`

NewRolesRoleMetadata instantiates a new RolesRoleMetadata object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolesRoleMetadataWithDefaults

`func NewRolesRoleMetadataWithDefaults() *RolesRoleMetadata`

NewRolesRoleMetadataWithDefaults instantiates a new RolesRoleMetadata object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *RolesRoleMetadata) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *RolesRoleMetadata) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *RolesRoleMetadata) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *RolesRoleMetadata) HasName() bool`

HasName returns a boolean if a field has been set.

### GetDomainType

`func (o *RolesRoleMetadata) GetDomainType() AuthorizationDomainType`

GetDomainType returns the DomainType field if non-nil, zero value otherwise.

### GetDomainTypeOk

`func (o *RolesRoleMetadata) GetDomainTypeOk() (*AuthorizationDomainType, bool)`

GetDomainTypeOk returns a tuple with the DomainType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainType

`func (o *RolesRoleMetadata) SetDomainType(v AuthorizationDomainType)`

SetDomainType sets DomainType field to given value.

### HasDomainType

`func (o *RolesRoleMetadata) HasDomainType() bool`

HasDomainType returns a boolean if a field has been set.

### GetDomainId

`func (o *RolesRoleMetadata) GetDomainId() string`

GetDomainId returns the DomainId field if non-nil, zero value otherwise.

### GetDomainIdOk

`func (o *RolesRoleMetadata) GetDomainIdOk() (*string, bool)`

GetDomainIdOk returns a tuple with the DomainId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainId

`func (o *RolesRoleMetadata) SetDomainId(v string)`

SetDomainId sets DomainId field to given value.

### HasDomainId

`func (o *RolesRoleMetadata) HasDomainId() bool`

HasDomainId returns a boolean if a field has been set.

### GetCreatedAt

`func (o *RolesRoleMetadata) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *RolesRoleMetadata) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *RolesRoleMetadata) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *RolesRoleMetadata) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *RolesRoleMetadata) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *RolesRoleMetadata) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *RolesRoleMetadata) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *RolesRoleMetadata) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


