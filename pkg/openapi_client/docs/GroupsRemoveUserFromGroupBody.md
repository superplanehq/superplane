# GroupsRemoveUserFromGroupBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DomainType** | Pointer to [**AuthorizationDomainType**](AuthorizationDomainType.md) |  | [optional] [default to AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_UNSPECIFIED]
**DomainId** | Pointer to **string** |  | [optional] 
**UserId** | Pointer to **string** |  | [optional] 
**UserEmail** | Pointer to **string** |  | [optional] 

## Methods

### NewGroupsRemoveUserFromGroupBody

`func NewGroupsRemoveUserFromGroupBody() *GroupsRemoveUserFromGroupBody`

NewGroupsRemoveUserFromGroupBody instantiates a new GroupsRemoveUserFromGroupBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGroupsRemoveUserFromGroupBodyWithDefaults

`func NewGroupsRemoveUserFromGroupBodyWithDefaults() *GroupsRemoveUserFromGroupBody`

NewGroupsRemoveUserFromGroupBodyWithDefaults instantiates a new GroupsRemoveUserFromGroupBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDomainType

`func (o *GroupsRemoveUserFromGroupBody) GetDomainType() AuthorizationDomainType`

GetDomainType returns the DomainType field if non-nil, zero value otherwise.

### GetDomainTypeOk

`func (o *GroupsRemoveUserFromGroupBody) GetDomainTypeOk() (*AuthorizationDomainType, bool)`

GetDomainTypeOk returns a tuple with the DomainType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainType

`func (o *GroupsRemoveUserFromGroupBody) SetDomainType(v AuthorizationDomainType)`

SetDomainType sets DomainType field to given value.

### HasDomainType

`func (o *GroupsRemoveUserFromGroupBody) HasDomainType() bool`

HasDomainType returns a boolean if a field has been set.

### GetDomainId

`func (o *GroupsRemoveUserFromGroupBody) GetDomainId() string`

GetDomainId returns the DomainId field if non-nil, zero value otherwise.

### GetDomainIdOk

`func (o *GroupsRemoveUserFromGroupBody) GetDomainIdOk() (*string, bool)`

GetDomainIdOk returns a tuple with the DomainId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainId

`func (o *GroupsRemoveUserFromGroupBody) SetDomainId(v string)`

SetDomainId sets DomainId field to given value.

### HasDomainId

`func (o *GroupsRemoveUserFromGroupBody) HasDomainId() bool`

HasDomainId returns a boolean if a field has been set.

### GetUserId

`func (o *GroupsRemoveUserFromGroupBody) GetUserId() string`

GetUserId returns the UserId field if non-nil, zero value otherwise.

### GetUserIdOk

`func (o *GroupsRemoveUserFromGroupBody) GetUserIdOk() (*string, bool)`

GetUserIdOk returns a tuple with the UserId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserId

`func (o *GroupsRemoveUserFromGroupBody) SetUserId(v string)`

SetUserId sets UserId field to given value.

### HasUserId

`func (o *GroupsRemoveUserFromGroupBody) HasUserId() bool`

HasUserId returns a boolean if a field has been set.

### GetUserEmail

`func (o *GroupsRemoveUserFromGroupBody) GetUserEmail() string`

GetUserEmail returns the UserEmail field if non-nil, zero value otherwise.

### GetUserEmailOk

`func (o *GroupsRemoveUserFromGroupBody) GetUserEmailOk() (*string, bool)`

GetUserEmailOk returns a tuple with the UserEmail field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserEmail

`func (o *GroupsRemoveUserFromGroupBody) SetUserEmail(v string)`

SetUserEmail sets UserEmail field to given value.

### HasUserEmail

`func (o *GroupsRemoveUserFromGroupBody) HasUserEmail() bool`

HasUserEmail returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


