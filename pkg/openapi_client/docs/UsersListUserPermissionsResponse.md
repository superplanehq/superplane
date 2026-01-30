# UsersListUserPermissionsResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**UserId** | Pointer to **string** |  | [optional] 
**DomainType** | Pointer to [**AuthorizationDomainType**](AuthorizationDomainType.md) |  | [optional] [default to AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_UNSPECIFIED]
**DomainId** | Pointer to **string** |  | [optional] 
**Permissions** | Pointer to [**[]AuthorizationPermission**](AuthorizationPermission.md) |  | [optional] 

## Methods

### NewUsersListUserPermissionsResponse

`func NewUsersListUserPermissionsResponse() *UsersListUserPermissionsResponse`

NewUsersListUserPermissionsResponse instantiates a new UsersListUserPermissionsResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUsersListUserPermissionsResponseWithDefaults

`func NewUsersListUserPermissionsResponseWithDefaults() *UsersListUserPermissionsResponse`

NewUsersListUserPermissionsResponseWithDefaults instantiates a new UsersListUserPermissionsResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetUserId

`func (o *UsersListUserPermissionsResponse) GetUserId() string`

GetUserId returns the UserId field if non-nil, zero value otherwise.

### GetUserIdOk

`func (o *UsersListUserPermissionsResponse) GetUserIdOk() (*string, bool)`

GetUserIdOk returns a tuple with the UserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserId

`func (o *UsersListUserPermissionsResponse) SetUserId(v string)`

SetUserId sets UserId field to given value.

### HasUserId

`func (o *UsersListUserPermissionsResponse) HasUserId() bool`

HasUserId returns a boolean if a field has been set.

### GetDomainType

`func (o *UsersListUserPermissionsResponse) GetDomainType() AuthorizationDomainType`

GetDomainType returns the DomainType field if non-nil, zero value otherwise.

### GetDomainTypeOk

`func (o *UsersListUserPermissionsResponse) GetDomainTypeOk() (*AuthorizationDomainType, bool)`

GetDomainTypeOk returns a tuple with the DomainType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainType

`func (o *UsersListUserPermissionsResponse) SetDomainType(v AuthorizationDomainType)`

SetDomainType sets DomainType field to given value.

### HasDomainType

`func (o *UsersListUserPermissionsResponse) HasDomainType() bool`

HasDomainType returns a boolean if a field has been set.

### GetDomainId

`func (o *UsersListUserPermissionsResponse) GetDomainId() string`

GetDomainId returns the DomainId field if non-nil, zero value otherwise.

### GetDomainIdOk

`func (o *UsersListUserPermissionsResponse) GetDomainIdOk() (*string, bool)`

GetDomainIdOk returns a tuple with the DomainId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainId

`func (o *UsersListUserPermissionsResponse) SetDomainId(v string)`

SetDomainId sets DomainId field to given value.

### HasDomainId

`func (o *UsersListUserPermissionsResponse) HasDomainId() bool`

HasDomainId returns a boolean if a field has been set.

### GetPermissions

`func (o *UsersListUserPermissionsResponse) GetPermissions() []AuthorizationPermission`

GetPermissions returns the Permissions field if non-nil, zero value otherwise.

### GetPermissionsOk

`func (o *UsersListUserPermissionsResponse) GetPermissionsOk() (*[]AuthorizationPermission, bool)`

GetPermissionsOk returns a tuple with the Permissions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPermissions

`func (o *UsersListUserPermissionsResponse) SetPermissions(v []AuthorizationPermission)`

SetPermissions sets Permissions field to given value.

### HasPermissions

`func (o *UsersListUserPermissionsResponse) HasPermissions() bool`

HasPermissions returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


