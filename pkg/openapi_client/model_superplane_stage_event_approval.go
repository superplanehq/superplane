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
	"time"
)

// checks if the SuperplaneStageEventApproval type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &SuperplaneStageEventApproval{}

// SuperplaneStageEventApproval struct for SuperplaneStageEventApproval
type SuperplaneStageEventApproval struct {
	ApprovedBy *string `json:"approvedBy,omitempty"`
	ApprovedAt *time.Time `json:"approvedAt,omitempty"`
}

// NewSuperplaneStageEventApproval instantiates a new SuperplaneStageEventApproval object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewSuperplaneStageEventApproval() *SuperplaneStageEventApproval {
	this := SuperplaneStageEventApproval{}
	return &this
}

// NewSuperplaneStageEventApprovalWithDefaults instantiates a new SuperplaneStageEventApproval object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewSuperplaneStageEventApprovalWithDefaults() *SuperplaneStageEventApproval {
	this := SuperplaneStageEventApproval{}
	return &this
}

// GetApprovedBy returns the ApprovedBy field value if set, zero value otherwise.
func (o *SuperplaneStageEventApproval) GetApprovedBy() string {
	if o == nil || IsNil(o.ApprovedBy) {
		var ret string
		return ret
	}
	return *o.ApprovedBy
}

// GetApprovedByOk returns a tuple with the ApprovedBy field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneStageEventApproval) GetApprovedByOk() (*string, bool) {
	if o == nil || IsNil(o.ApprovedBy) {
		return nil, false
	}
	return o.ApprovedBy, true
}

// HasApprovedBy returns a boolean if a field has been set.
func (o *SuperplaneStageEventApproval) HasApprovedBy() bool {
	if o != nil && !IsNil(o.ApprovedBy) {
		return true
	}

	return false
}

// SetApprovedBy gets a reference to the given string and assigns it to the ApprovedBy field.
func (o *SuperplaneStageEventApproval) SetApprovedBy(v string) {
	o.ApprovedBy = &v
}

// GetApprovedAt returns the ApprovedAt field value if set, zero value otherwise.
func (o *SuperplaneStageEventApproval) GetApprovedAt() time.Time {
	if o == nil || IsNil(o.ApprovedAt) {
		var ret time.Time
		return ret
	}
	return *o.ApprovedAt
}

// GetApprovedAtOk returns a tuple with the ApprovedAt field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneStageEventApproval) GetApprovedAtOk() (*time.Time, bool) {
	if o == nil || IsNil(o.ApprovedAt) {
		return nil, false
	}
	return o.ApprovedAt, true
}

// HasApprovedAt returns a boolean if a field has been set.
func (o *SuperplaneStageEventApproval) HasApprovedAt() bool {
	if o != nil && !IsNil(o.ApprovedAt) {
		return true
	}

	return false
}

// SetApprovedAt gets a reference to the given time.Time and assigns it to the ApprovedAt field.
func (o *SuperplaneStageEventApproval) SetApprovedAt(v time.Time) {
	o.ApprovedAt = &v
}

func (o SuperplaneStageEventApproval) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o SuperplaneStageEventApproval) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.ApprovedBy) {
		toSerialize["approvedBy"] = o.ApprovedBy
	}
	if !IsNil(o.ApprovedAt) {
		toSerialize["approvedAt"] = o.ApprovedAt
	}
	return toSerialize, nil
}

type NullableSuperplaneStageEventApproval struct {
	value *SuperplaneStageEventApproval
	isSet bool
}

func (v NullableSuperplaneStageEventApproval) Get() *SuperplaneStageEventApproval {
	return v.value
}

func (v *NullableSuperplaneStageEventApproval) Set(val *SuperplaneStageEventApproval) {
	v.value = val
	v.isSet = true
}

func (v NullableSuperplaneStageEventApproval) IsSet() bool {
	return v.isSet
}

func (v *NullableSuperplaneStageEventApproval) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableSuperplaneStageEventApproval(val *SuperplaneStageEventApproval) *NullableSuperplaneStageEventApproval {
	return &NullableSuperplaneStageEventApproval{value: val, isSet: true}
}

func (v NullableSuperplaneStageEventApproval) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableSuperplaneStageEventApproval) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


