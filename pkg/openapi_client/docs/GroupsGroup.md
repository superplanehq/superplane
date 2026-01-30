# GroupsGroup

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Metadata** | Pointer to [**GroupsGroupMetadata**](GroupsGroupMetadata.md) |  | [optional] 
**Spec** | Pointer to [**GroupsGroupSpec**](GroupsGroupSpec.md) |  | [optional] 
**Status** | Pointer to [**GroupsGroupStatus**](GroupsGroupStatus.md) |  | [optional] 

## Methods

### NewGroupsGroup

`func NewGroupsGroup() *GroupsGroup`

NewGroupsGroup instantiates a new GroupsGroup object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGroupsGroupWithDefaults

`func NewGroupsGroupWithDefaults() *GroupsGroup`

NewGroupsGroupWithDefaults instantiates a new GroupsGroup object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMetadata

`func (o *GroupsGroup) GetMetadata() GroupsGroupMetadata`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *GroupsGroup) GetMetadataOk() (*GroupsGroupMetadata, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *GroupsGroup) SetMetadata(v GroupsGroupMetadata)`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *GroupsGroup) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetSpec

`func (o *GroupsGroup) GetSpec() GroupsGroupSpec`

GetSpec returns the Spec field if non-nil, zero value otherwise.

### GetSpecOk

`func (o *GroupsGroup) GetSpecOk() (*GroupsGroupSpec, bool)`

GetSpecOk returns a tuple with the Spec field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSpec

`func (o *GroupsGroup) SetSpec(v GroupsGroupSpec)`

SetSpec sets Spec field to given value.

### HasSpec

`func (o *GroupsGroup) HasSpec() bool`

HasSpec returns a boolean if a field has been set.

### GetStatus

`func (o *GroupsGroup) GetStatus() GroupsGroupStatus`

GetStatus returns the Status field if non-nil, zero value otherwise.

### GetStatusOk

`func (o *GroupsGroup) GetStatusOk() (*GroupsGroupStatus, bool)`

GetStatusOk returns a tuple with the Status field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStatus

`func (o *GroupsGroup) SetStatus(v GroupsGroupStatus)`

SetStatus sets Status field to given value.

### HasStatus

`func (o *GroupsGroup) HasStatus() bool`

HasStatus returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


