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

// checks if the SuperplaneStage type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &SuperplaneStage{}

// SuperplaneStage struct for SuperplaneStage
type SuperplaneStage struct {
	Metadata *SuperplaneStageMetadata `json:"metadata,omitempty"`
	Spec *SuperplaneStageSpec `json:"spec,omitempty"`
}

// NewSuperplaneStage instantiates a new SuperplaneStage object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewSuperplaneStage() *SuperplaneStage {
	this := SuperplaneStage{}
	return &this
}

// NewSuperplaneStageWithDefaults instantiates a new SuperplaneStage object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewSuperplaneStageWithDefaults() *SuperplaneStage {
	this := SuperplaneStage{}
	return &this
}

// GetMetadata returns the Metadata field value if set, zero value otherwise.
func (o *SuperplaneStage) GetMetadata() SuperplaneStageMetadata {
	if o == nil || IsNil(o.Metadata) {
		var ret SuperplaneStageMetadata
		return ret
	}
	return *o.Metadata
}

// GetMetadataOk returns a tuple with the Metadata field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneStage) GetMetadataOk() (*SuperplaneStageMetadata, bool) {
	if o == nil || IsNil(o.Metadata) {
		return nil, false
	}
	return o.Metadata, true
}

// HasMetadata returns a boolean if a field has been set.
func (o *SuperplaneStage) HasMetadata() bool {
	if o != nil && !IsNil(o.Metadata) {
		return true
	}

	return false
}

// SetMetadata gets a reference to the given SuperplaneStageMetadata and assigns it to the Metadata field.
func (o *SuperplaneStage) SetMetadata(v SuperplaneStageMetadata) {
	o.Metadata = &v
}

// GetSpec returns the Spec field value if set, zero value otherwise.
func (o *SuperplaneStage) GetSpec() SuperplaneStageSpec {
	if o == nil || IsNil(o.Spec) {
		var ret SuperplaneStageSpec
		return ret
	}
	return *o.Spec
}

// GetSpecOk returns a tuple with the Spec field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneStage) GetSpecOk() (*SuperplaneStageSpec, bool) {
	if o == nil || IsNil(o.Spec) {
		return nil, false
	}
	return o.Spec, true
}

// HasSpec returns a boolean if a field has been set.
func (o *SuperplaneStage) HasSpec() bool {
	if o != nil && !IsNil(o.Spec) {
		return true
	}

	return false
}

// SetSpec gets a reference to the given SuperplaneStageSpec and assigns it to the Spec field.
func (o *SuperplaneStage) SetSpec(v SuperplaneStageSpec) {
	o.Spec = &v
}

func (o SuperplaneStage) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o SuperplaneStage) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.Metadata) {
		toSerialize["metadata"] = o.Metadata
	}
	if !IsNil(o.Spec) {
		toSerialize["spec"] = o.Spec
	}
	return toSerialize, nil
}

type NullableSuperplaneStage struct {
	value *SuperplaneStage
	isSet bool
}

func (v NullableSuperplaneStage) Get() *SuperplaneStage {
	return v.value
}

func (v *NullableSuperplaneStage) Set(val *SuperplaneStage) {
	v.value = val
	v.isSet = true
}

func (v NullableSuperplaneStage) IsSet() bool {
	return v.isSet
}

func (v *NullableSuperplaneStage) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableSuperplaneStage(val *SuperplaneStage) *NullableSuperplaneStage {
	return &NullableSuperplaneStage{value: val, isSet: true}
}

func (v NullableSuperplaneStage) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableSuperplaneStage) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


