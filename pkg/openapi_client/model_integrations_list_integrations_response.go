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

// checks if the IntegrationsListIntegrationsResponse type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &IntegrationsListIntegrationsResponse{}

// IntegrationsListIntegrationsResponse struct for IntegrationsListIntegrationsResponse
type IntegrationsListIntegrationsResponse struct {
	Integrations []IntegrationsIntegration `json:"integrations,omitempty"`
}

// NewIntegrationsListIntegrationsResponse instantiates a new IntegrationsListIntegrationsResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewIntegrationsListIntegrationsResponse() *IntegrationsListIntegrationsResponse {
	this := IntegrationsListIntegrationsResponse{}
	return &this
}

// NewIntegrationsListIntegrationsResponseWithDefaults instantiates a new IntegrationsListIntegrationsResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewIntegrationsListIntegrationsResponseWithDefaults() *IntegrationsListIntegrationsResponse {
	this := IntegrationsListIntegrationsResponse{}
	return &this
}

// GetIntegrations returns the Integrations field value if set, zero value otherwise.
func (o *IntegrationsListIntegrationsResponse) GetIntegrations() []IntegrationsIntegration {
	if o == nil || IsNil(o.Integrations) {
		var ret []IntegrationsIntegration
		return ret
	}
	return o.Integrations
}

// GetIntegrationsOk returns a tuple with the Integrations field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IntegrationsListIntegrationsResponse) GetIntegrationsOk() ([]IntegrationsIntegration, bool) {
	if o == nil || IsNil(o.Integrations) {
		return nil, false
	}
	return o.Integrations, true
}

// HasIntegrations returns a boolean if a field has been set.
func (o *IntegrationsListIntegrationsResponse) HasIntegrations() bool {
	if o != nil && !IsNil(o.Integrations) {
		return true
	}

	return false
}

// SetIntegrations gets a reference to the given []IntegrationsIntegration and assigns it to the Integrations field.
func (o *IntegrationsListIntegrationsResponse) SetIntegrations(v []IntegrationsIntegration) {
	o.Integrations = v
}

func (o IntegrationsListIntegrationsResponse) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o IntegrationsListIntegrationsResponse) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.Integrations) {
		toSerialize["integrations"] = o.Integrations
	}
	return toSerialize, nil
}

type NullableIntegrationsListIntegrationsResponse struct {
	value *IntegrationsListIntegrationsResponse
	isSet bool
}

func (v NullableIntegrationsListIntegrationsResponse) Get() *IntegrationsListIntegrationsResponse {
	return v.value
}

func (v *NullableIntegrationsListIntegrationsResponse) Set(val *IntegrationsListIntegrationsResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableIntegrationsListIntegrationsResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableIntegrationsListIntegrationsResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableIntegrationsListIntegrationsResponse(val *IntegrationsListIntegrationsResponse) *NullableIntegrationsListIntegrationsResponse {
	return &NullableIntegrationsListIntegrationsResponse{value: val, isSet: true}
}

func (v NullableIntegrationsListIntegrationsResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableIntegrationsListIntegrationsResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


