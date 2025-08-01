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

// checks if the SuperplaneCreateStageResponse type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &SuperplaneCreateStageResponse{}

// SuperplaneCreateStageResponse struct for SuperplaneCreateStageResponse
type SuperplaneCreateStageResponse struct {
	Stage *SuperplaneStage `json:"stage,omitempty"`
}

// NewSuperplaneCreateStageResponse instantiates a new SuperplaneCreateStageResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewSuperplaneCreateStageResponse() *SuperplaneCreateStageResponse {
	this := SuperplaneCreateStageResponse{}
	return &this
}

// NewSuperplaneCreateStageResponseWithDefaults instantiates a new SuperplaneCreateStageResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewSuperplaneCreateStageResponseWithDefaults() *SuperplaneCreateStageResponse {
	this := SuperplaneCreateStageResponse{}
	return &this
}

// GetStage returns the Stage field value if set, zero value otherwise.
func (o *SuperplaneCreateStageResponse) GetStage() SuperplaneStage {
	if o == nil || IsNil(o.Stage) {
		var ret SuperplaneStage
		return ret
	}
	return *o.Stage
}

// GetStageOk returns a tuple with the Stage field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneCreateStageResponse) GetStageOk() (*SuperplaneStage, bool) {
	if o == nil || IsNil(o.Stage) {
		return nil, false
	}
	return o.Stage, true
}

// HasStage returns a boolean if a field has been set.
func (o *SuperplaneCreateStageResponse) HasStage() bool {
	if o != nil && !IsNil(o.Stage) {
		return true
	}

	return false
}

// SetStage gets a reference to the given SuperplaneStage and assigns it to the Stage field.
func (o *SuperplaneCreateStageResponse) SetStage(v SuperplaneStage) {
	o.Stage = &v
}

func (o SuperplaneCreateStageResponse) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o SuperplaneCreateStageResponse) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.Stage) {
		toSerialize["stage"] = o.Stage
	}
	return toSerialize, nil
}

type NullableSuperplaneCreateStageResponse struct {
	value *SuperplaneCreateStageResponse
	isSet bool
}

func (v NullableSuperplaneCreateStageResponse) Get() *SuperplaneCreateStageResponse {
	return v.value
}

func (v *NullableSuperplaneCreateStageResponse) Set(val *SuperplaneCreateStageResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableSuperplaneCreateStageResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableSuperplaneCreateStageResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableSuperplaneCreateStageResponse(val *SuperplaneCreateStageResponse) *NullableSuperplaneCreateStageResponse {
	return &NullableSuperplaneCreateStageResponse{value: val, isSet: true}
}

func (v NullableSuperplaneCreateStageResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableSuperplaneCreateStageResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


