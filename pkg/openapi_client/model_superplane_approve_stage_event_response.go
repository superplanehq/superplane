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

// checks if the SuperplaneApproveStageEventResponse type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &SuperplaneApproveStageEventResponse{}

// SuperplaneApproveStageEventResponse struct for SuperplaneApproveStageEventResponse
type SuperplaneApproveStageEventResponse struct {
	Event *SuperplaneStageEvent `json:"event,omitempty"`
}

// NewSuperplaneApproveStageEventResponse instantiates a new SuperplaneApproveStageEventResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewSuperplaneApproveStageEventResponse() *SuperplaneApproveStageEventResponse {
	this := SuperplaneApproveStageEventResponse{}
	return &this
}

// NewSuperplaneApproveStageEventResponseWithDefaults instantiates a new SuperplaneApproveStageEventResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewSuperplaneApproveStageEventResponseWithDefaults() *SuperplaneApproveStageEventResponse {
	this := SuperplaneApproveStageEventResponse{}
	return &this
}

// GetEvent returns the Event field value if set, zero value otherwise.
func (o *SuperplaneApproveStageEventResponse) GetEvent() SuperplaneStageEvent {
	if o == nil || IsNil(o.Event) {
		var ret SuperplaneStageEvent
		return ret
	}
	return *o.Event
}

// GetEventOk returns a tuple with the Event field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneApproveStageEventResponse) GetEventOk() (*SuperplaneStageEvent, bool) {
	if o == nil || IsNil(o.Event) {
		return nil, false
	}
	return o.Event, true
}

// HasEvent returns a boolean if a field has been set.
func (o *SuperplaneApproveStageEventResponse) HasEvent() bool {
	if o != nil && !IsNil(o.Event) {
		return true
	}

	return false
}

// SetEvent gets a reference to the given SuperplaneStageEvent and assigns it to the Event field.
func (o *SuperplaneApproveStageEventResponse) SetEvent(v SuperplaneStageEvent) {
	o.Event = &v
}

func (o SuperplaneApproveStageEventResponse) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o SuperplaneApproveStageEventResponse) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.Event) {
		toSerialize["event"] = o.Event
	}
	return toSerialize, nil
}

type NullableSuperplaneApproveStageEventResponse struct {
	value *SuperplaneApproveStageEventResponse
	isSet bool
}

func (v NullableSuperplaneApproveStageEventResponse) Get() *SuperplaneApproveStageEventResponse {
	return v.value
}

func (v *NullableSuperplaneApproveStageEventResponse) Set(val *SuperplaneApproveStageEventResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableSuperplaneApproveStageEventResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableSuperplaneApproveStageEventResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableSuperplaneApproveStageEventResponse(val *SuperplaneApproveStageEventResponse) *NullableSuperplaneApproveStageEventResponse {
	return &NullableSuperplaneApproveStageEventResponse{value: val, isSet: true}
}

func (v NullableSuperplaneApproveStageEventResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableSuperplaneApproveStageEventResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


