# AuthorizationPermission

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Resource** | Pointer to **string** |  | [optional] 
**Action** | Pointer to **string** |  | [optional] 
**DomainType** | Pointer to [**AuthorizationDomainType**](AuthorizationDomainType.md) |  | [optional] [default to AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_UNSPECIFIED]

## Methods

### NewAuthorizationPermission

`func NewAuthorizationPermission() *AuthorizationPermission`

NewAuthorizationPermission instantiates a new AuthorizationPermission object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAuthorizationPermissionWithDefaults

`func NewAuthorizationPermissionWithDefaults() *AuthorizationPermission`

NewAuthorizationPermissionWithDefaults instantiates a new AuthorizationPermission object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetResource

`func (o *AuthorizationPermission) GetResource() string`

GetResource returns the Resource field if non-nil, zero value otherwise.

### GetResourceOk

`func (o *AuthorizationPermission) GetResourceOk() (*string, bool)`

GetResourceOk returns a tuple with the Resource field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResource

`func (o *AuthorizationPermission) SetResource(v string)`

SetResource sets Resource field to given value.

### HasResource

`func (o *AuthorizationPermission) HasResource() bool`

HasResource returns a boolean if a field has been set.

### GetAction

`func (o *AuthorizationPermission) GetAction() string`

GetAction returns the Action field if non-nil, zero value otherwise.

### GetActionOk

`func (o *AuthorizationPermission) GetActionOk() (*string, bool)`

GetActionOk returns a tuple with the Action field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAction

`func (o *AuthorizationPermission) SetAction(v string)`

SetAction sets Action field to given value.

### HasAction

`func (o *AuthorizationPermission) HasAction() bool`

HasAction returns a boolean if a field has been set.

### GetDomainType

`func (o *AuthorizationPermission) GetDomainType() AuthorizationDomainType`

GetDomainType returns the DomainType field if non-nil, zero value otherwise.

### GetDomainTypeOk

`func (o *AuthorizationPermission) GetDomainTypeOk() (*AuthorizationDomainType, bool)`

GetDomainTypeOk returns a tuple with the DomainType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainType

`func (o *AuthorizationPermission) SetDomainType(v AuthorizationDomainType)`

SetDomainType sets DomainType field to given value.

### HasDomainType

`func (o *AuthorizationPermission) HasDomainType() bool`

HasDomainType returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


