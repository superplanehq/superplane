# GroupsUpdateGroupBody

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DomainType** | Pointer to [**AuthorizationDomainType**](AuthorizationDomainType.md) |  | [optional] [default to AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_UNSPECIFIED]
**DomainId** | Pointer to **string** |  | [optional] 
**Group** | Pointer to [**GroupsGroup**](GroupsGroup.md) |  | [optional] 

## Methods

### NewGroupsUpdateGroupBody

`func NewGroupsUpdateGroupBody() *GroupsUpdateGroupBody`

NewGroupsUpdateGroupBody instantiates a new GroupsUpdateGroupBody object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGroupsUpdateGroupBodyWithDefaults

`func NewGroupsUpdateGroupBodyWithDefaults() *GroupsUpdateGroupBody`

NewGroupsUpdateGroupBodyWithDefaults instantiates a new GroupsUpdateGroupBody object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDomainType

`func (o *GroupsUpdateGroupBody) GetDomainType() AuthorizationDomainType`

GetDomainType returns the DomainType field if non-nil, zero value otherwise.

### GetDomainTypeOk

`func (o *GroupsUpdateGroupBody) GetDomainTypeOk() (*AuthorizationDomainType, bool)`

GetDomainTypeOk returns a tuple with the DomainType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainType

`func (o *GroupsUpdateGroupBody) SetDomainType(v AuthorizationDomainType)`

SetDomainType sets DomainType field to given value.

### HasDomainType

`func (o *GroupsUpdateGroupBody) HasDomainType() bool`

HasDomainType returns a boolean if a field has been set.

### GetDomainId

`func (o *GroupsUpdateGroupBody) GetDomainId() string`

GetDomainId returns the DomainId field if non-nil, zero value otherwise.

### GetDomainIdOk

`func (o *GroupsUpdateGroupBody) GetDomainIdOk() (*string, bool)`

GetDomainIdOk returns a tuple with the DomainId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainId

`func (o *GroupsUpdateGroupBody) SetDomainId(v string)`

SetDomainId sets DomainId field to given value.

### HasDomainId

`func (o *GroupsUpdateGroupBody) HasDomainId() bool`

HasDomainId returns a boolean if a field has been set.

### GetGroup

`func (o *GroupsUpdateGroupBody) GetGroup() GroupsGroup`

GetGroup returns the Group field if non-nil, zero value otherwise.

### GetGroupOk

`func (o *GroupsUpdateGroupBody) GetGroupOk() (*GroupsGroup, bool)`

GetGroupOk returns a tuple with the Group field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGroup

`func (o *GroupsUpdateGroupBody) SetGroup(v GroupsGroup)`

SetGroup sets Group field to given value.

### HasGroup

`func (o *GroupsUpdateGroupBody) HasGroup() bool`

HasGroup returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


