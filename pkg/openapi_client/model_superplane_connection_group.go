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

// checks if the SuperplaneConnectionGroup type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &SuperplaneConnectionGroup{}

// SuperplaneConnectionGroup struct for SuperplaneConnectionGroup
type SuperplaneConnectionGroup struct {
	Metadata *SuperplaneConnectionGroupMetadata `json:"metadata,omitempty"`
	Spec *SuperplaneConnectionGroupSpec `json:"spec,omitempty"`
}

// NewSuperplaneConnectionGroup instantiates a new SuperplaneConnectionGroup object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewSuperplaneConnectionGroup() *SuperplaneConnectionGroup {
	this := SuperplaneConnectionGroup{}
	return &this
}

// NewSuperplaneConnectionGroupWithDefaults instantiates a new SuperplaneConnectionGroup object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewSuperplaneConnectionGroupWithDefaults() *SuperplaneConnectionGroup {
	this := SuperplaneConnectionGroup{}
	return &this
}

// GetMetadata returns the Metadata field value if set, zero value otherwise.
func (o *SuperplaneConnectionGroup) GetMetadata() SuperplaneConnectionGroupMetadata {
	if o == nil || IsNil(o.Metadata) {
		var ret SuperplaneConnectionGroupMetadata
		return ret
	}
	return *o.Metadata
}

// GetMetadataOk returns a tuple with the Metadata field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneConnectionGroup) GetMetadataOk() (*SuperplaneConnectionGroupMetadata, bool) {
	if o == nil || IsNil(o.Metadata) {
		return nil, false
	}
	return o.Metadata, true
}

// HasMetadata returns a boolean if a field has been set.
func (o *SuperplaneConnectionGroup) HasMetadata() bool {
	if o != nil && !IsNil(o.Metadata) {
		return true
	}

	return false
}

// SetMetadata gets a reference to the given SuperplaneConnectionGroupMetadata and assigns it to the Metadata field.
func (o *SuperplaneConnectionGroup) SetMetadata(v SuperplaneConnectionGroupMetadata) {
	o.Metadata = &v
}

// GetSpec returns the Spec field value if set, zero value otherwise.
func (o *SuperplaneConnectionGroup) GetSpec() SuperplaneConnectionGroupSpec {
	if o == nil || IsNil(o.Spec) {
		var ret SuperplaneConnectionGroupSpec
		return ret
	}
	return *o.Spec
}

// GetSpecOk returns a tuple with the Spec field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneConnectionGroup) GetSpecOk() (*SuperplaneConnectionGroupSpec, bool) {
	if o == nil || IsNil(o.Spec) {
		return nil, false
	}
	return o.Spec, true
}

// HasSpec returns a boolean if a field has been set.
func (o *SuperplaneConnectionGroup) HasSpec() bool {
	if o != nil && !IsNil(o.Spec) {
		return true
	}

	return false
}

// SetSpec gets a reference to the given SuperplaneConnectionGroupSpec and assigns it to the Spec field.
func (o *SuperplaneConnectionGroup) SetSpec(v SuperplaneConnectionGroupSpec) {
	o.Spec = &v
}

func (o SuperplaneConnectionGroup) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o SuperplaneConnectionGroup) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.Metadata) {
		toSerialize["metadata"] = o.Metadata
	}
	if !IsNil(o.Spec) {
		toSerialize["spec"] = o.Spec
	}
	return toSerialize, nil
}

type NullableSuperplaneConnectionGroup struct {
	value *SuperplaneConnectionGroup
	isSet bool
}

func (v NullableSuperplaneConnectionGroup) Get() *SuperplaneConnectionGroup {
	return v.value
}

func (v *NullableSuperplaneConnectionGroup) Set(val *SuperplaneConnectionGroup) {
	v.value = val
	v.isSet = true
}

func (v NullableSuperplaneConnectionGroup) IsSet() bool {
	return v.isSet
}

func (v *NullableSuperplaneConnectionGroup) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableSuperplaneConnectionGroup(val *SuperplaneConnectionGroup) *NullableSuperplaneConnectionGroup {
	return &NullableSuperplaneConnectionGroup{value: val, isSet: true}
}

func (v NullableSuperplaneConnectionGroup) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableSuperplaneConnectionGroup) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


