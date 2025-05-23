/*
Superplane API

API for the Superplane service

API version: 1.0
Contact: support@superplane.com
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package openapi_client

import (
	"encoding/json"
)

// checks if the ConnectionHeaderFilter type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &ConnectionHeaderFilter{}

// ConnectionHeaderFilter struct for ConnectionHeaderFilter
type ConnectionHeaderFilter struct {
	Expression *string `json:"expression,omitempty"`
}

// NewConnectionHeaderFilter instantiates a new ConnectionHeaderFilter object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewConnectionHeaderFilter() *ConnectionHeaderFilter {
	this := ConnectionHeaderFilter{}
	return &this
}

// NewConnectionHeaderFilterWithDefaults instantiates a new ConnectionHeaderFilter object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewConnectionHeaderFilterWithDefaults() *ConnectionHeaderFilter {
	this := ConnectionHeaderFilter{}
	return &this
}

// GetExpression returns the Expression field value if set, zero value otherwise.
func (o *ConnectionHeaderFilter) GetExpression() string {
	if o == nil || IsNil(o.Expression) {
		var ret string
		return ret
	}
	return *o.Expression
}

// GetExpressionOk returns a tuple with the Expression field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *ConnectionHeaderFilter) GetExpressionOk() (*string, bool) {
	if o == nil || IsNil(o.Expression) {
		return nil, false
	}
	return o.Expression, true
}

// HasExpression returns a boolean if a field has been set.
func (o *ConnectionHeaderFilter) HasExpression() bool {
	if o != nil && !IsNil(o.Expression) {
		return true
	}

	return false
}

// SetExpression gets a reference to the given string and assigns it to the Expression field.
func (o *ConnectionHeaderFilter) SetExpression(v string) {
	o.Expression = &v
}

func (o ConnectionHeaderFilter) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o ConnectionHeaderFilter) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.Expression) {
		toSerialize["expression"] = o.Expression
	}
	return toSerialize, nil
}

type NullableConnectionHeaderFilter struct {
	value *ConnectionHeaderFilter
	isSet bool
}

func (v NullableConnectionHeaderFilter) Get() *ConnectionHeaderFilter {
	return v.value
}

func (v *NullableConnectionHeaderFilter) Set(val *ConnectionHeaderFilter) {
	v.value = val
	v.isSet = true
}

func (v NullableConnectionHeaderFilter) IsSet() bool {
	return v.isSet
}

func (v *NullableConnectionHeaderFilter) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableConnectionHeaderFilter(val *ConnectionHeaderFilter) *NullableConnectionHeaderFilter {
	return &NullableConnectionHeaderFilter{value: val, isSet: true}
}

func (v NullableConnectionHeaderFilter) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableConnectionHeaderFilter) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


