# UsersUserRoleAssignment

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RoleName** | Pointer to **string** |  | [optional] 
**RoleDisplayName** | Pointer to **string** |  | [optional] 
**RoleDescription** | Pointer to **string** |  | [optional] 
**DomainType** | Pointer to [**AuthorizationDomainType**](AuthorizationDomainType.md) |  | [optional] [default to AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_UNSPECIFIED]
**DomainId** | Pointer to **string** |  | [optional] 
**AssignedAt** | Pointer to **time.Time** |  | [optional] 

## Methods

### NewUsersUserRoleAssignment

`func NewUsersUserRoleAssignment() *UsersUserRoleAssignment`

NewUsersUserRoleAssignment instantiates a new UsersUserRoleAssignment object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUsersUserRoleAssignmentWithDefaults

`func NewUsersUserRoleAssignmentWithDefaults() *UsersUserRoleAssignment`

NewUsersUserRoleAssignmentWithDefaults instantiates a new UsersUserRoleAssignment object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRoleName

`func (o *UsersUserRoleAssignment) GetRoleName() string`

GetRoleName returns the RoleName field if non-nil, zero value otherwise.

### GetRoleNameOk

`func (o *UsersUserRoleAssignment) GetRoleNameOk() (*string, bool)`

GetRoleNameOk returns a tuple with the RoleName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRoleName

`func (o *UsersUserRoleAssignment) SetRoleName(v string)`

SetRoleName sets RoleName field to given value.

### HasRoleName

`func (o *UsersUserRoleAssignment) HasRoleName() bool`

HasRoleName returns a boolean if a field has been set.

### GetRoleDisplayName

`func (o *UsersUserRoleAssignment) GetRoleDisplayName() string`

GetRoleDisplayName returns the RoleDisplayName field if non-nil, zero value otherwise.

### GetRoleDisplayNameOk

`func (o *UsersUserRoleAssignment) GetRoleDisplayNameOk() (*string, bool)`

GetRoleDisplayNameOk returns a tuple with the RoleDisplayName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRoleDisplayName

`func (o *UsersUserRoleAssignment) SetRoleDisplayName(v string)`

SetRoleDisplayName sets RoleDisplayName field to given value.

### HasRoleDisplayName

`func (o *UsersUserRoleAssignment) HasRoleDisplayName() bool`

HasRoleDisplayName returns a boolean if a field has been set.

### GetRoleDescription

`func (o *UsersUserRoleAssignment) GetRoleDescription() string`

GetRoleDescription returns the RoleDescription field if non-nil, zero value otherwise.

### GetRoleDescriptionOk

`func (o *UsersUserRoleAssignment) GetRoleDescriptionOk() (*string, bool)`

GetRoleDescriptionOk returns a tuple with the RoleDescription field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRoleDescription

`func (o *UsersUserRoleAssignment) SetRoleDescription(v string)`

SetRoleDescription sets RoleDescription field to given value.

### HasRoleDescription

`func (o *UsersUserRoleAssignment) HasRoleDescription() bool`

HasRoleDescription returns a boolean if a field has been set.

### GetDomainType

`func (o *UsersUserRoleAssignment) GetDomainType() AuthorizationDomainType`

GetDomainType returns the DomainType field if non-nil, zero value otherwise.

### GetDomainTypeOk

`func (o *UsersUserRoleAssignment) GetDomainTypeOk() (*AuthorizationDomainType, bool)`

GetDomainTypeOk returns a tuple with the DomainType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainType

`func (o *UsersUserRoleAssignment) SetDomainType(v AuthorizationDomainType)`

SetDomainType sets DomainType field to given value.

### HasDomainType

`func (o *UsersUserRoleAssignment) HasDomainType() bool`

HasDomainType returns a boolean if a field has been set.

### GetDomainId

`func (o *UsersUserRoleAssignment) GetDomainId() string`

GetDomainId returns the DomainId field if non-nil, zero value otherwise.

### GetDomainIdOk

`func (o *UsersUserRoleAssignment) GetDomainIdOk() (*string, bool)`

GetDomainIdOk returns a tuple with the DomainId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainId

`func (o *UsersUserRoleAssignment) SetDomainId(v string)`

SetDomainId sets DomainId field to given value.

### HasDomainId

`func (o *UsersUserRoleAssignment) HasDomainId() bool`

HasDomainId returns a boolean if a field has been set.

### GetAssignedAt

`func (o *UsersUserRoleAssignment) GetAssignedAt() time.Time`

GetAssignedAt returns the AssignedAt field if non-nil, zero value otherwise.

### GetAssignedAtOk

`func (o *UsersUserRoleAssignment) GetAssignedAtOk() (*time.Time, bool)`

GetAssignedAtOk returns a tuple with the AssignedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAssignedAt

`func (o *UsersUserRoleAssignment) SetAssignedAt(v time.Time)`

SetAssignedAt sets AssignedAt field to given value.

### HasAssignedAt

`func (o *UsersUserRoleAssignment) HasAssignedAt() bool`

HasAssignedAt returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


