/*
Superplane Organizations API

API for managing organizations in the Superplane service

API version: 1.0
Contact: support@superplane.com
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package openapi_client

import (
	"encoding/json"
)

// checks if the GroupsDescribeGroupResponse type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &GroupsDescribeGroupResponse{}

// GroupsDescribeGroupResponse struct for GroupsDescribeGroupResponse
type GroupsDescribeGroupResponse struct {
	Group *GroupsGroup `json:"group,omitempty"`
}

// NewGroupsDescribeGroupResponse instantiates a new GroupsDescribeGroupResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewGroupsDescribeGroupResponse() *GroupsDescribeGroupResponse {
	this := GroupsDescribeGroupResponse{}
	return &this
}

// NewGroupsDescribeGroupResponseWithDefaults instantiates a new GroupsDescribeGroupResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewGroupsDescribeGroupResponseWithDefaults() *GroupsDescribeGroupResponse {
	this := GroupsDescribeGroupResponse{}
	return &this
}

// GetGroup returns the Group field value if set, zero value otherwise.
func (o *GroupsDescribeGroupResponse) GetGroup() GroupsGroup {
	if o == nil || IsNil(o.Group) {
		var ret GroupsGroup
		return ret
	}
	return *o.Group
}

// GetGroupOk returns a tuple with the Group field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GroupsDescribeGroupResponse) GetGroupOk() (*GroupsGroup, bool) {
	if o == nil || IsNil(o.Group) {
		return nil, false
	}
	return o.Group, true
}

// HasGroup returns a boolean if a field has been set.
func (o *GroupsDescribeGroupResponse) HasGroup() bool {
	if o != nil && !IsNil(o.Group) {
		return true
	}

	return false
}

// SetGroup gets a reference to the given GroupsGroup and assigns it to the Group field.
func (o *GroupsDescribeGroupResponse) SetGroup(v GroupsGroup) {
	o.Group = &v
}

func (o GroupsDescribeGroupResponse) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o GroupsDescribeGroupResponse) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.Group) {
		toSerialize["group"] = o.Group
	}
	return toSerialize, nil
}

type NullableGroupsDescribeGroupResponse struct {
	value *GroupsDescribeGroupResponse
	isSet bool
}

func (v NullableGroupsDescribeGroupResponse) Get() *GroupsDescribeGroupResponse {
	return v.value
}

func (v *NullableGroupsDescribeGroupResponse) Set(val *GroupsDescribeGroupResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableGroupsDescribeGroupResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableGroupsDescribeGroupResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableGroupsDescribeGroupResponse(val *GroupsDescribeGroupResponse) *NullableGroupsDescribeGroupResponse {
	return &NullableGroupsDescribeGroupResponse{value: val, isSet: true}
}

func (v NullableGroupsDescribeGroupResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableGroupsDescribeGroupResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


