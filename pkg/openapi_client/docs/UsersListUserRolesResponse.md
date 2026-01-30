# UsersListUserRolesResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**UserId** | Pointer to **string** |  | [optional] 
**DomainType** | Pointer to [**AuthorizationDomainType**](AuthorizationDomainType.md) |  | [optional] [default to AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_UNSPECIFIED]
**DomainId** | Pointer to **string** |  | [optional] 
**Roles** | Pointer to [**[]RolesRole**](RolesRole.md) |  | [optional] 

## Methods

### NewUsersListUserRolesResponse

`func NewUsersListUserRolesResponse() *UsersListUserRolesResponse`

NewUsersListUserRolesResponse instantiates a new UsersListUserRolesResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUsersListUserRolesResponseWithDefaults

`func NewUsersListUserRolesResponseWithDefaults() *UsersListUserRolesResponse`

NewUsersListUserRolesResponseWithDefaults instantiates a new UsersListUserRolesResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetUserId

`func (o *UsersListUserRolesResponse) GetUserId() string`

GetUserId returns the UserId field if non-nil, zero value otherwise.

### GetUserIdOk

`func (o *UsersListUserRolesResponse) GetUserIdOk() (*string, bool)`

GetUserIdOk returns a tuple with the UserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserId

`func (o *UsersListUserRolesResponse) SetUserId(v string)`

SetUserId sets UserId field to given value.

### HasUserId

`func (o *UsersListUserRolesResponse) HasUserId() bool`

HasUserId returns a boolean if a field has been set.

### GetDomainType

`func (o *UsersListUserRolesResponse) GetDomainType() AuthorizationDomainType`

GetDomainType returns the DomainType field if non-nil, zero value otherwise.

### GetDomainTypeOk

`func (o *UsersListUserRolesResponse) GetDomainTypeOk() (*AuthorizationDomainType, bool)`

GetDomainTypeOk returns a tuple with the DomainType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainType

`func (o *UsersListUserRolesResponse) SetDomainType(v AuthorizationDomainType)`

SetDomainType sets DomainType field to given value.

### HasDomainType

`func (o *UsersListUserRolesResponse) HasDomainType() bool`

HasDomainType returns a boolean if a field has been set.

### GetDomainId

`func (o *UsersListUserRolesResponse) GetDomainId() string`

GetDomainId returns the DomainId field if non-nil, zero value otherwise.

### GetDomainIdOk

`func (o *UsersListUserRolesResponse) GetDomainIdOk() (*string, bool)`

GetDomainIdOk returns a tuple with the DomainId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainId

`func (o *UsersListUserRolesResponse) SetDomainId(v string)`

SetDomainId sets DomainId field to given value.

### HasDomainId

`func (o *UsersListUserRolesResponse) HasDomainId() bool`

HasDomainId returns a boolean if a field has been set.

### GetRoles

`func (o *UsersListUserRolesResponse) GetRoles() []RolesRole`

GetRoles returns the Roles field if non-nil, zero value otherwise.

### GetRolesOk

`func (o *UsersListUserRolesResponse) GetRolesOk() (*[]RolesRole, bool)`

GetRolesOk returns a tuple with the Roles field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRoles

`func (o *UsersListUserRolesResponse) SetRoles(v []RolesRole)`

SetRoles sets Roles field to given value.

### HasRoles

`func (o *UsersListUserRolesResponse) HasRoles() bool`

HasRoles returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


