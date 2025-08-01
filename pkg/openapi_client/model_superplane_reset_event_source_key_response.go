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

// checks if the SuperplaneResetEventSourceKeyResponse type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &SuperplaneResetEventSourceKeyResponse{}

// SuperplaneResetEventSourceKeyResponse struct for SuperplaneResetEventSourceKeyResponse
type SuperplaneResetEventSourceKeyResponse struct {
	EventSource *SuperplaneEventSource `json:"eventSource,omitempty"`
	Key *string `json:"key,omitempty"`
}

// NewSuperplaneResetEventSourceKeyResponse instantiates a new SuperplaneResetEventSourceKeyResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewSuperplaneResetEventSourceKeyResponse() *SuperplaneResetEventSourceKeyResponse {
	this := SuperplaneResetEventSourceKeyResponse{}
	return &this
}

// NewSuperplaneResetEventSourceKeyResponseWithDefaults instantiates a new SuperplaneResetEventSourceKeyResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewSuperplaneResetEventSourceKeyResponseWithDefaults() *SuperplaneResetEventSourceKeyResponse {
	this := SuperplaneResetEventSourceKeyResponse{}
	return &this
}

// GetEventSource returns the EventSource field value if set, zero value otherwise.
func (o *SuperplaneResetEventSourceKeyResponse) GetEventSource() SuperplaneEventSource {
	if o == nil || IsNil(o.EventSource) {
		var ret SuperplaneEventSource
		return ret
	}
	return *o.EventSource
}

// GetEventSourceOk returns a tuple with the EventSource field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneResetEventSourceKeyResponse) GetEventSourceOk() (*SuperplaneEventSource, bool) {
	if o == nil || IsNil(o.EventSource) {
		return nil, false
	}
	return o.EventSource, true
}

// HasEventSource returns a boolean if a field has been set.
func (o *SuperplaneResetEventSourceKeyResponse) HasEventSource() bool {
	if o != nil && !IsNil(o.EventSource) {
		return true
	}

	return false
}

// SetEventSource gets a reference to the given SuperplaneEventSource and assigns it to the EventSource field.
func (o *SuperplaneResetEventSourceKeyResponse) SetEventSource(v SuperplaneEventSource) {
	o.EventSource = &v
}

// GetKey returns the Key field value if set, zero value otherwise.
func (o *SuperplaneResetEventSourceKeyResponse) GetKey() string {
	if o == nil || IsNil(o.Key) {
		var ret string
		return ret
	}
	return *o.Key
}

// GetKeyOk returns a tuple with the Key field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneResetEventSourceKeyResponse) GetKeyOk() (*string, bool) {
	if o == nil || IsNil(o.Key) {
		return nil, false
	}
	return o.Key, true
}

// HasKey returns a boolean if a field has been set.
func (o *SuperplaneResetEventSourceKeyResponse) HasKey() bool {
	if o != nil && !IsNil(o.Key) {
		return true
	}

	return false
}

// SetKey gets a reference to the given string and assigns it to the Key field.
func (o *SuperplaneResetEventSourceKeyResponse) SetKey(v string) {
	o.Key = &v
}

func (o SuperplaneResetEventSourceKeyResponse) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o SuperplaneResetEventSourceKeyResponse) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.EventSource) {
		toSerialize["eventSource"] = o.EventSource
	}
	if !IsNil(o.Key) {
		toSerialize["key"] = o.Key
	}
	return toSerialize, nil
}

type NullableSuperplaneResetEventSourceKeyResponse struct {
	value *SuperplaneResetEventSourceKeyResponse
	isSet bool
}

func (v NullableSuperplaneResetEventSourceKeyResponse) Get() *SuperplaneResetEventSourceKeyResponse {
	return v.value
}

func (v *NullableSuperplaneResetEventSourceKeyResponse) Set(val *SuperplaneResetEventSourceKeyResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableSuperplaneResetEventSourceKeyResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableSuperplaneResetEventSourceKeyResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableSuperplaneResetEventSourceKeyResponse(val *SuperplaneResetEventSourceKeyResponse) *NullableSuperplaneResetEventSourceKeyResponse {
	return &NullableSuperplaneResetEventSourceKeyResponse{value: val, isSet: true}
}

func (v NullableSuperplaneResetEventSourceKeyResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableSuperplaneResetEventSourceKeyResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


