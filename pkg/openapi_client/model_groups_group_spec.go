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

// checks if the GroupsGroupSpec type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &GroupsGroupSpec{}

// GroupsGroupSpec struct for GroupsGroupSpec
type GroupsGroupSpec struct {
	Role *string `json:"role,omitempty"`
	DisplayName *string `json:"displayName,omitempty"`
	Description *string `json:"description,omitempty"`
}

// NewGroupsGroupSpec instantiates a new GroupsGroupSpec object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewGroupsGroupSpec() *GroupsGroupSpec {
	this := GroupsGroupSpec{}
	return &this
}

// NewGroupsGroupSpecWithDefaults instantiates a new GroupsGroupSpec object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewGroupsGroupSpecWithDefaults() *GroupsGroupSpec {
	this := GroupsGroupSpec{}
	return &this
}

// GetRole returns the Role field value if set, zero value otherwise.
func (o *GroupsGroupSpec) GetRole() string {
	if o == nil || IsNil(o.Role) {
		var ret string
		return ret
	}
	return *o.Role
}

// GetRoleOk returns a tuple with the Role field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GroupsGroupSpec) GetRoleOk() (*string, bool) {
	if o == nil || IsNil(o.Role) {
		return nil, false
	}
	return o.Role, true
}

// HasRole returns a boolean if a field has been set.
func (o *GroupsGroupSpec) HasRole() bool {
	if o != nil && !IsNil(o.Role) {
		return true
	}

	return false
}

// SetRole gets a reference to the given string and assigns it to the Role field.
func (o *GroupsGroupSpec) SetRole(v string) {
	o.Role = &v
}

// GetDisplayName returns the DisplayName field value if set, zero value otherwise.
func (o *GroupsGroupSpec) GetDisplayName() string {
	if o == nil || IsNil(o.DisplayName) {
		var ret string
		return ret
	}
	return *o.DisplayName
}

// GetDisplayNameOk returns a tuple with the DisplayName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GroupsGroupSpec) GetDisplayNameOk() (*string, bool) {
	if o == nil || IsNil(o.DisplayName) {
		return nil, false
	}
	return o.DisplayName, true
}

// HasDisplayName returns a boolean if a field has been set.
func (o *GroupsGroupSpec) HasDisplayName() bool {
	if o != nil && !IsNil(o.DisplayName) {
		return true
	}

	return false
}

// SetDisplayName gets a reference to the given string and assigns it to the DisplayName field.
func (o *GroupsGroupSpec) SetDisplayName(v string) {
	o.DisplayName = &v
}

// GetDescription returns the Description field value if set, zero value otherwise.
func (o *GroupsGroupSpec) GetDescription() string {
	if o == nil || IsNil(o.Description) {
		var ret string
		return ret
	}
	return *o.Description
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GroupsGroupSpec) GetDescriptionOk() (*string, bool) {
	if o == nil || IsNil(o.Description) {
		return nil, false
	}
	return o.Description, true
}

// HasDescription returns a boolean if a field has been set.
func (o *GroupsGroupSpec) HasDescription() bool {
	if o != nil && !IsNil(o.Description) {
		return true
	}

	return false
}

// SetDescription gets a reference to the given string and assigns it to the Description field.
func (o *GroupsGroupSpec) SetDescription(v string) {
	o.Description = &v
}

func (o GroupsGroupSpec) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o GroupsGroupSpec) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.Role) {
		toSerialize["role"] = o.Role
	}
	if !IsNil(o.DisplayName) {
		toSerialize["displayName"] = o.DisplayName
	}
	if !IsNil(o.Description) {
		toSerialize["description"] = o.Description
	}
	return toSerialize, nil
}

type NullableGroupsGroupSpec struct {
	value *GroupsGroupSpec
	isSet bool
}

func (v NullableGroupsGroupSpec) Get() *GroupsGroupSpec {
	return v.value
}

func (v *NullableGroupsGroupSpec) Set(val *GroupsGroupSpec) {
	v.value = val
	v.isSet = true
}

func (v NullableGroupsGroupSpec) IsSet() bool {
	return v.isSet
}

func (v *NullableGroupsGroupSpec) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableGroupsGroupSpec(val *GroupsGroupSpec) *NullableGroupsGroupSpec {
	return &NullableGroupsGroupSpec{value: val, isSet: true}
}

func (v NullableGroupsGroupSpec) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableGroupsGroupSpec) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


