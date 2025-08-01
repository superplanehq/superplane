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

// checks if the SuperplaneInputDefinition type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &SuperplaneInputDefinition{}

// SuperplaneInputDefinition struct for SuperplaneInputDefinition
type SuperplaneInputDefinition struct {
	Name *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// NewSuperplaneInputDefinition instantiates a new SuperplaneInputDefinition object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewSuperplaneInputDefinition() *SuperplaneInputDefinition {
	this := SuperplaneInputDefinition{}
	return &this
}

// NewSuperplaneInputDefinitionWithDefaults instantiates a new SuperplaneInputDefinition object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewSuperplaneInputDefinitionWithDefaults() *SuperplaneInputDefinition {
	this := SuperplaneInputDefinition{}
	return &this
}

// GetName returns the Name field value if set, zero value otherwise.
func (o *SuperplaneInputDefinition) GetName() string {
	if o == nil || IsNil(o.Name) {
		var ret string
		return ret
	}
	return *o.Name
}

// GetNameOk returns a tuple with the Name field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneInputDefinition) GetNameOk() (*string, bool) {
	if o == nil || IsNil(o.Name) {
		return nil, false
	}
	return o.Name, true
}

// HasName returns a boolean if a field has been set.
func (o *SuperplaneInputDefinition) HasName() bool {
	if o != nil && !IsNil(o.Name) {
		return true
	}

	return false
}

// SetName gets a reference to the given string and assigns it to the Name field.
func (o *SuperplaneInputDefinition) SetName(v string) {
	o.Name = &v
}

// GetDescription returns the Description field value if set, zero value otherwise.
func (o *SuperplaneInputDefinition) GetDescription() string {
	if o == nil || IsNil(o.Description) {
		var ret string
		return ret
	}
	return *o.Description
}

// GetDescriptionOk returns a tuple with the Description field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneInputDefinition) GetDescriptionOk() (*string, bool) {
	if o == nil || IsNil(o.Description) {
		return nil, false
	}
	return o.Description, true
}

// HasDescription returns a boolean if a field has been set.
func (o *SuperplaneInputDefinition) HasDescription() bool {
	if o != nil && !IsNil(o.Description) {
		return true
	}

	return false
}

// SetDescription gets a reference to the given string and assigns it to the Description field.
func (o *SuperplaneInputDefinition) SetDescription(v string) {
	o.Description = &v
}

func (o SuperplaneInputDefinition) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o SuperplaneInputDefinition) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.Name) {
		toSerialize["name"] = o.Name
	}
	if !IsNil(o.Description) {
		toSerialize["description"] = o.Description
	}
	return toSerialize, nil
}

type NullableSuperplaneInputDefinition struct {
	value *SuperplaneInputDefinition
	isSet bool
}

func (v NullableSuperplaneInputDefinition) Get() *SuperplaneInputDefinition {
	return v.value
}

func (v *NullableSuperplaneInputDefinition) Set(val *SuperplaneInputDefinition) {
	v.value = val
	v.isSet = true
}

func (v NullableSuperplaneInputDefinition) IsSet() bool {
	return v.isSet
}

func (v *NullableSuperplaneInputDefinition) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableSuperplaneInputDefinition(val *SuperplaneInputDefinition) *NullableSuperplaneInputDefinition {
	return &NullableSuperplaneInputDefinition{value: val, isSet: true}
}

func (v NullableSuperplaneInputDefinition) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableSuperplaneInputDefinition) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


