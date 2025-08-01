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

// checks if the SuperplaneDescribeCanvasResponse type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &SuperplaneDescribeCanvasResponse{}

// SuperplaneDescribeCanvasResponse struct for SuperplaneDescribeCanvasResponse
type SuperplaneDescribeCanvasResponse struct {
	Canvas *SuperplaneCanvas `json:"canvas,omitempty"`
}

// NewSuperplaneDescribeCanvasResponse instantiates a new SuperplaneDescribeCanvasResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewSuperplaneDescribeCanvasResponse() *SuperplaneDescribeCanvasResponse {
	this := SuperplaneDescribeCanvasResponse{}
	return &this
}

// NewSuperplaneDescribeCanvasResponseWithDefaults instantiates a new SuperplaneDescribeCanvasResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewSuperplaneDescribeCanvasResponseWithDefaults() *SuperplaneDescribeCanvasResponse {
	this := SuperplaneDescribeCanvasResponse{}
	return &this
}

// GetCanvas returns the Canvas field value if set, zero value otherwise.
func (o *SuperplaneDescribeCanvasResponse) GetCanvas() SuperplaneCanvas {
	if o == nil || IsNil(o.Canvas) {
		var ret SuperplaneCanvas
		return ret
	}
	return *o.Canvas
}

// GetCanvasOk returns a tuple with the Canvas field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneDescribeCanvasResponse) GetCanvasOk() (*SuperplaneCanvas, bool) {
	if o == nil || IsNil(o.Canvas) {
		return nil, false
	}
	return o.Canvas, true
}

// HasCanvas returns a boolean if a field has been set.
func (o *SuperplaneDescribeCanvasResponse) HasCanvas() bool {
	if o != nil && !IsNil(o.Canvas) {
		return true
	}

	return false
}

// SetCanvas gets a reference to the given SuperplaneCanvas and assigns it to the Canvas field.
func (o *SuperplaneDescribeCanvasResponse) SetCanvas(v SuperplaneCanvas) {
	o.Canvas = &v
}

func (o SuperplaneDescribeCanvasResponse) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o SuperplaneDescribeCanvasResponse) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.Canvas) {
		toSerialize["canvas"] = o.Canvas
	}
	return toSerialize, nil
}

type NullableSuperplaneDescribeCanvasResponse struct {
	value *SuperplaneDescribeCanvasResponse
	isSet bool
}

func (v NullableSuperplaneDescribeCanvasResponse) Get() *SuperplaneDescribeCanvasResponse {
	return v.value
}

func (v *NullableSuperplaneDescribeCanvasResponse) Set(val *SuperplaneDescribeCanvasResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableSuperplaneDescribeCanvasResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableSuperplaneDescribeCanvasResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableSuperplaneDescribeCanvasResponse(val *SuperplaneDescribeCanvasResponse) *NullableSuperplaneDescribeCanvasResponse {
	return &NullableSuperplaneDescribeCanvasResponse{value: val, isSet: true}
}

func (v NullableSuperplaneDescribeCanvasResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableSuperplaneDescribeCanvasResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


