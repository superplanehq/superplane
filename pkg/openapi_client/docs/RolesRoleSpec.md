# RolesRoleSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DisplayName** | Pointer to **string** |  | [optional] 
**Description** | Pointer to **string** |  | [optional] 
**Permissions** | Pointer to [**[]AuthorizationPermission**](AuthorizationPermission.md) |  | [optional] 
**InheritedRole** | Pointer to [**RolesRole**](RolesRole.md) |  | [optional] 

## Methods

### NewRolesRoleSpec

`func NewRolesRoleSpec() *RolesRoleSpec`

NewRolesRoleSpec instantiates a new RolesRoleSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewRolesRoleSpecWithDefaults

`func NewRolesRoleSpecWithDefaults() *RolesRoleSpec`

NewRolesRoleSpecWithDefaults instantiates a new RolesRoleSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDisplayName

`func (o *RolesRoleSpec) GetDisplayName() string`

GetDisplayName returns the DisplayName field if non-nil, zero value otherwise.

### GetDisplayNameOk

`func (o *RolesRoleSpec) GetDisplayNameOk() (*string, bool)`

GetDisplayNameOk returns a tuple with the DisplayName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDisplayName

`func (o *RolesRoleSpec) SetDisplayName(v string)`

SetDisplayName sets DisplayName field to given value.

### HasDisplayName

`func (o *RolesRoleSpec) HasDisplayName() bool`

HasDisplayName returns a boolean if a field has been set.

### GetDescription

`func (o *RolesRoleSpec) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *RolesRoleSpec) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *RolesRoleSpec) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *RolesRoleSpec) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetPermissions

`func (o *RolesRoleSpec) GetPermissions() []AuthorizationPermission`

GetPermissions returns the Permissions field if non-nil, zero value otherwise.

### GetPermissionsOk

`func (o *RolesRoleSpec) GetPermissionsOk() (*[]AuthorizationPermission, bool)`

GetPermissionsOk returns a tuple with the Permissions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPermissions

`func (o *RolesRoleSpec) SetPermissions(v []AuthorizationPermission)`

SetPermissions sets Permissions field to given value.

### HasPermissions

`func (o *RolesRoleSpec) HasPermissions() bool`

HasPermissions returns a boolean if a field has been set.

### GetInheritedRole

`func (o *RolesRoleSpec) GetInheritedRole() RolesRole`

GetInheritedRole returns the InheritedRole field if non-nil, zero value otherwise.

### GetInheritedRoleOk

`func (o *RolesRoleSpec) GetInheritedRoleOk() (*RolesRole, bool)`

GetInheritedRoleOk returns a tuple with the InheritedRole field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInheritedRole

`func (o *RolesRoleSpec) SetInheritedRole(v RolesRole)`

SetInheritedRole sets InheritedRole field to given value.

### HasInheritedRole

`func (o *RolesRoleSpec) HasInheritedRole() bool`

HasInheritedRole returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


