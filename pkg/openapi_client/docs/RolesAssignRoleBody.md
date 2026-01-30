# RolesAssignRoleBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DomainType** | Pointer to [**AuthorizationDomainType**](AuthorizationDomainType.md) |  | [optional] [default to AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_UNSPECIFIED]
**DomainId** | Pointer to **string** |  | [optional] 
**UserId** | Pointer to **string** |  | [optional] 
**UserEmail** | Pointer to **string** |  | [optional] 

## Methods

### NewRolesAssignRoleBody

`func NewRolesAssignRoleBody() *RolesAssignRoleBody`

NewRolesAssignRoleBody instantiates a new RolesAssignRoleBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolesAssignRoleBodyWithDefaults

`func NewRolesAssignRoleBodyWithDefaults() *RolesAssignRoleBody`

NewRolesAssignRoleBodyWithDefaults instantiates a new RolesAssignRoleBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDomainType

`func (o *RolesAssignRoleBody) GetDomainType() AuthorizationDomainType`

GetDomainType returns the DomainType field if non-nil, zero value otherwise.

### GetDomainTypeOk

`func (o *RolesAssignRoleBody) GetDomainTypeOk() (*AuthorizationDomainType, bool)`

GetDomainTypeOk returns a tuple with the DomainType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainType

`func (o *RolesAssignRoleBody) SetDomainType(v AuthorizationDomainType)`

SetDomainType sets DomainType field to given value.

### HasDomainType

`func (o *RolesAssignRoleBody) HasDomainType() bool`

HasDomainType returns a boolean if a field has been set.

### GetDomainId

`func (o *RolesAssignRoleBody) GetDomainId() string`

GetDomainId returns the DomainId field if non-nil, zero value otherwise.

### GetDomainIdOk

`func (o *RolesAssignRoleBody) GetDomainIdOk() (*string, bool)`

GetDomainIdOk returns a tuple with the DomainId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainId

`func (o *RolesAssignRoleBody) SetDomainId(v string)`

SetDomainId sets DomainId field to given value.

### HasDomainId

`func (o *RolesAssignRoleBody) HasDomainId() bool`

HasDomainId returns a boolean if a field has been set.

### GetUserId

`func (o *RolesAssignRoleBody) GetUserId() string`

GetUserId returns the UserId field if non-nil, zero value otherwise.

### GetUserIdOk

`func (o *RolesAssignRoleBody) GetUserIdOk() (*string, bool)`

GetUserIdOk returns a tuple with the UserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserId

`func (o *RolesAssignRoleBody) SetUserId(v string)`

SetUserId sets UserId field to given value.

### HasUserId

`func (o *RolesAssignRoleBody) HasUserId() bool`

HasUserId returns a boolean if a field has been set.

### GetUserEmail

`func (o *RolesAssignRoleBody) GetUserEmail() string`

GetUserEmail returns the UserEmail field if non-nil, zero value otherwise.

### GetUserEmailOk

`func (o *RolesAssignRoleBody) GetUserEmailOk() (*string, bool)`

GetUserEmailOk returns a tuple with the UserEmail field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserEmail

`func (o *RolesAssignRoleBody) SetUserEmail(v string)`

SetUserEmail sets UserEmail field to given value.

### HasUserEmail

`func (o *RolesAssignRoleBody) HasUserEmail() bool`

HasUserEmail returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


