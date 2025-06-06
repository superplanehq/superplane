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

// checks if the SuperplaneDescribeSecretResponse type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &SuperplaneDescribeSecretResponse{}

// SuperplaneDescribeSecretResponse struct for SuperplaneDescribeSecretResponse
type SuperplaneDescribeSecretResponse struct {
	Secret *SuperplaneSecret `json:"secret,omitempty"`
}

// NewSuperplaneDescribeSecretResponse instantiates a new SuperplaneDescribeSecretResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewSuperplaneDescribeSecretResponse() *SuperplaneDescribeSecretResponse {
	this := SuperplaneDescribeSecretResponse{}
	return &this
}

// NewSuperplaneDescribeSecretResponseWithDefaults instantiates a new SuperplaneDescribeSecretResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewSuperplaneDescribeSecretResponseWithDefaults() *SuperplaneDescribeSecretResponse {
	this := SuperplaneDescribeSecretResponse{}
	return &this
}

// GetSecret returns the Secret field value if set, zero value otherwise.
func (o *SuperplaneDescribeSecretResponse) GetSecret() SuperplaneSecret {
	if o == nil || IsNil(o.Secret) {
		var ret SuperplaneSecret
		return ret
	}
	return *o.Secret
}

// GetSecretOk returns a tuple with the Secret field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneDescribeSecretResponse) GetSecretOk() (*SuperplaneSecret, bool) {
	if o == nil || IsNil(o.Secret) {
		return nil, false
	}
	return o.Secret, true
}

// HasSecret returns a boolean if a field has been set.
func (o *SuperplaneDescribeSecretResponse) HasSecret() bool {
	if o != nil && !IsNil(o.Secret) {
		return true
	}

	return false
}

// SetSecret gets a reference to the given SuperplaneSecret and assigns it to the Secret field.
func (o *SuperplaneDescribeSecretResponse) SetSecret(v SuperplaneSecret) {
	o.Secret = &v
}

func (o SuperplaneDescribeSecretResponse) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o SuperplaneDescribeSecretResponse) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.Secret) {
		toSerialize["secret"] = o.Secret
	}
	return toSerialize, nil
}

type NullableSuperplaneDescribeSecretResponse struct {
	value *SuperplaneDescribeSecretResponse
	isSet bool
}

func (v NullableSuperplaneDescribeSecretResponse) Get() *SuperplaneDescribeSecretResponse {
	return v.value
}

func (v *NullableSuperplaneDescribeSecretResponse) Set(val *SuperplaneDescribeSecretResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableSuperplaneDescribeSecretResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableSuperplaneDescribeSecretResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableSuperplaneDescribeSecretResponse(val *SuperplaneDescribeSecretResponse) *NullableSuperplaneDescribeSecretResponse {
	return &NullableSuperplaneDescribeSecretResponse{value: val, isSet: true}
}

func (v NullableSuperplaneDescribeSecretResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableSuperplaneDescribeSecretResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


