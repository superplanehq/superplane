# RolesUpdateRoleBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DomainType** | Pointer to [**AuthorizationDomainType**](AuthorizationDomainType.md) |  | [optional] [default to AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_UNSPECIFIED]
**DomainId** | Pointer to **string** |  | [optional] 
**Role** | Pointer to [**RolesRole**](RolesRole.md) |  | [optional] 

## Methods

### NewRolesUpdateRoleBody

`func NewRolesUpdateRoleBody() *RolesUpdateRoleBody`

NewRolesUpdateRoleBody instantiates a new RolesUpdateRoleBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolesUpdateRoleBodyWithDefaults

`func NewRolesUpdateRoleBodyWithDefaults() *RolesUpdateRoleBody`

NewRolesUpdateRoleBodyWithDefaults instantiates a new RolesUpdateRoleBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDomainType

`func (o *RolesUpdateRoleBody) GetDomainType() AuthorizationDomainType`

GetDomainType returns the DomainType field if non-nil, zero value otherwise.

### GetDomainTypeOk

`func (o *RolesUpdateRoleBody) GetDomainTypeOk() (*AuthorizationDomainType, bool)`

GetDomainTypeOk returns a tuple with the DomainType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainType

`func (o *RolesUpdateRoleBody) SetDomainType(v AuthorizationDomainType)`

SetDomainType sets DomainType field to given value.

### HasDomainType

`func (o *RolesUpdateRoleBody) HasDomainType() bool`

HasDomainType returns a boolean if a field has been set.

### GetDomainId

`func (o *RolesUpdateRoleBody) GetDomainId() string`

GetDomainId returns the DomainId field if non-nil, zero value otherwise.

### GetDomainIdOk

`func (o *RolesUpdateRoleBody) GetDomainIdOk() (*string, bool)`

GetDomainIdOk returns a tuple with the DomainId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainId

`func (o *RolesUpdateRoleBody) SetDomainId(v string)`

SetDomainId sets DomainId field to given value.

### HasDomainId

`func (o *RolesUpdateRoleBody) HasDomainId() bool`

HasDomainId returns a boolean if a field has been set.

### GetRole

`func (o *RolesUpdateRoleBody) GetRole() RolesRole`

GetRole returns the Role field if non-nil, zero value otherwise.

### GetRoleOk

`func (o *RolesUpdateRoleBody) GetRoleOk() (*RolesRole, bool)`

GetRoleOk returns a tuple with the Role field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRole

`func (o *RolesUpdateRoleBody) SetRole(v RolesRole)`

SetRole sets Role field to given value.

### HasRole

`func (o *RolesUpdateRoleBody) HasRole() bool`

HasRole returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


