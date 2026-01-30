# GroupsListGroupUsersResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Users** | Pointer to [**[]SuperplaneUsersUser**](SuperplaneUsersUser.md) |  | [optional] 
**Group** | Pointer to [**GroupsGroup**](GroupsGroup.md) |  | [optional] 

## Methods

### NewGroupsListGroupUsersResponse

`func NewGroupsListGroupUsersResponse() *GroupsListGroupUsersResponse`

NewGroupsListGroupUsersResponse instantiates a new GroupsListGroupUsersResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGroupsListGroupUsersResponseWithDefaults

`func NewGroupsListGroupUsersResponseWithDefaults() *GroupsListGroupUsersResponse`

NewGroupsListGroupUsersResponseWithDefaults instantiates a new GroupsListGroupUsersResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetUsers

`func (o *GroupsListGroupUsersResponse) GetUsers() []SuperplaneUsersUser`

GetUsers returns the Users field if non-nil, zero value otherwise.

### GetUsersOk

`func (o *GroupsListGroupUsersResponse) GetUsersOk() (*[]SuperplaneUsersUser, bool)`

GetUsersOk returns a tuple with the Users field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUsers

`func (o *GroupsListGroupUsersResponse) SetUsers(v []SuperplaneUsersUser)`

SetUsers sets Users field to given value.

### HasUsers

`func (o *GroupsListGroupUsersResponse) HasUsers() bool`

HasUsers returns a boolean if a field has been set.

### GetGroup

`func (o *GroupsListGroupUsersResponse) GetGroup() GroupsGroup`

GetGroup returns the Group field if non-nil, zero value otherwise.

### GetGroupOk

`func (o *GroupsListGroupUsersResponse) GetGroupOk() (*GroupsGroup, bool)`

GetGroupOk returns a tuple with the Group field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGroup

`func (o *GroupsListGroupUsersResponse) SetGroup(v GroupsGroup)`

SetGroup sets Group field to given value.

### HasGroup

`func (o *GroupsListGroupUsersResponse) HasGroup() bool`

HasGroup returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


