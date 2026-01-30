# GroupsCreateGroupRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DomainType** | Pointer to [**AuthorizationDomainType**](AuthorizationDomainType.md) |  | [optional] [default to AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_UNSPECIFIED]
**DomainId** | Pointer to **string** |  | [optional] 
**Group** | Pointer to [**GroupsGroup**](GroupsGroup.md) |  | [optional] 

## Methods

### NewGroupsCreateGroupRequest

`func NewGroupsCreateGroupRequest() *GroupsCreateGroupRequest`

NewGroupsCreateGroupRequest instantiates a new GroupsCreateGroupRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGroupsCreateGroupRequestWithDefaults

`func NewGroupsCreateGroupRequestWithDefaults() *GroupsCreateGroupRequest`

NewGroupsCreateGroupRequestWithDefaults instantiates a new GroupsCreateGroupRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDomainType

`func (o *GroupsCreateGroupRequest) GetDomainType() AuthorizationDomainType`

GetDomainType returns the DomainType field if non-nil, zero value otherwise.

### GetDomainTypeOk

`func (o *GroupsCreateGroupRequest) GetDomainTypeOk() (*AuthorizationDomainType, bool)`

GetDomainTypeOk returns a tuple with the DomainType field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainType

`func (o *GroupsCreateGroupRequest) SetDomainType(v AuthorizationDomainType)`

SetDomainType sets DomainType field to given value.

### HasDomainType

`func (o *GroupsCreateGroupRequest) HasDomainType() bool`

HasDomainType returns a boolean if a field has been set.

### GetDomainId

`func (o *GroupsCreateGroupRequest) GetDomainId() string`

GetDomainId returns the DomainId field if non-nil, zero value otherwise.

### GetDomainIdOk

`func (o *GroupsCreateGroupRequest) GetDomainIdOk() (*string, bool)`

GetDomainIdOk returns a tuple with the DomainId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDomainId

`func (o *GroupsCreateGroupRequest) SetDomainId(v string)`

SetDomainId sets DomainId field to given value.

### HasDomainId

`func (o *GroupsCreateGroupRequest) HasDomainId() bool`

HasDomainId returns a boolean if a field has been set.

### GetGroup

`func (o *GroupsCreateGroupRequest) GetGroup() GroupsGroup`

GetGroup returns the Group field if non-nil, zero value otherwise.

### GetGroupOk

`func (o *GroupsCreateGroupRequest) GetGroupOk() (*GroupsGroup, bool)`

GetGroupOk returns a tuple with the Group field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGroup

`func (o *GroupsCreateGroupRequest) SetGroup(v GroupsGroup)`

SetGroup sets Group field to given value.

### HasGroup

`func (o *GroupsCreateGroupRequest) HasGroup() bool`

HasGroup returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


