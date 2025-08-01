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

// checks if the SuperplaneDataFilter type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &SuperplaneDataFilter{}

// SuperplaneDataFilter struct for SuperplaneDataFilter
type SuperplaneDataFilter struct {
	Expression *string `json:"expression,omitempty"`
}

// NewSuperplaneDataFilter instantiates a new SuperplaneDataFilter object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewSuperplaneDataFilter() *SuperplaneDataFilter {
	this := SuperplaneDataFilter{}
	return &this
}

// NewSuperplaneDataFilterWithDefaults instantiates a new SuperplaneDataFilter object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewSuperplaneDataFilterWithDefaults() *SuperplaneDataFilter {
	this := SuperplaneDataFilter{}
	return &this
}

// GetExpression returns the Expression field value if set, zero value otherwise.
func (o *SuperplaneDataFilter) GetExpression() string {
	if o == nil || IsNil(o.Expression) {
		var ret string
		return ret
	}
	return *o.Expression
}

// GetExpressionOk returns a tuple with the Expression field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneDataFilter) GetExpressionOk() (*string, bool) {
	if o == nil || IsNil(o.Expression) {
		return nil, false
	}
	return o.Expression, true
}

// HasExpression returns a boolean if a field has been set.
func (o *SuperplaneDataFilter) HasExpression() bool {
	if o != nil && !IsNil(o.Expression) {
		return true
	}

	return false
}

// SetExpression gets a reference to the given string and assigns it to the Expression field.
func (o *SuperplaneDataFilter) SetExpression(v string) {
	o.Expression = &v
}

func (o SuperplaneDataFilter) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o SuperplaneDataFilter) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.Expression) {
		toSerialize["expression"] = o.Expression
	}
	return toSerialize, nil
}

type NullableSuperplaneDataFilter struct {
	value *SuperplaneDataFilter
	isSet bool
}

func (v NullableSuperplaneDataFilter) Get() *SuperplaneDataFilter {
	return v.value
}

func (v *NullableSuperplaneDataFilter) Set(val *SuperplaneDataFilter) {
	v.value = val
	v.isSet = true
}

func (v NullableSuperplaneDataFilter) IsSet() bool {
	return v.isSet
}

func (v *NullableSuperplaneDataFilter) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableSuperplaneDataFilter(val *SuperplaneDataFilter) *NullableSuperplaneDataFilter {
	return &NullableSuperplaneDataFilter{value: val, isSet: true}
}

func (v NullableSuperplaneDataFilter) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableSuperplaneDataFilter) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


