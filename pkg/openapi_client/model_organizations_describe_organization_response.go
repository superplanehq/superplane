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

// checks if the OrganizationsDescribeOrganizationResponse type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &OrganizationsDescribeOrganizationResponse{}

// OrganizationsDescribeOrganizationResponse struct for OrganizationsDescribeOrganizationResponse
type OrganizationsDescribeOrganizationResponse struct {
	Organization *OrganizationsOrganization `json:"organization,omitempty"`
}

// NewOrganizationsDescribeOrganizationResponse instantiates a new OrganizationsDescribeOrganizationResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewOrganizationsDescribeOrganizationResponse() *OrganizationsDescribeOrganizationResponse {
	this := OrganizationsDescribeOrganizationResponse{}
	return &this
}

// NewOrganizationsDescribeOrganizationResponseWithDefaults instantiates a new OrganizationsDescribeOrganizationResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewOrganizationsDescribeOrganizationResponseWithDefaults() *OrganizationsDescribeOrganizationResponse {
	this := OrganizationsDescribeOrganizationResponse{}
	return &this
}

// GetOrganization returns the Organization field value if set, zero value otherwise.
func (o *OrganizationsDescribeOrganizationResponse) GetOrganization() OrganizationsOrganization {
	if o == nil || IsNil(o.Organization) {
		var ret OrganizationsOrganization
		return ret
	}
	return *o.Organization
}

// GetOrganizationOk returns a tuple with the Organization field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *OrganizationsDescribeOrganizationResponse) GetOrganizationOk() (*OrganizationsOrganization, bool) {
	if o == nil || IsNil(o.Organization) {
		return nil, false
	}
	return o.Organization, true
}

// HasOrganization returns a boolean if a field has been set.
func (o *OrganizationsDescribeOrganizationResponse) HasOrganization() bool {
	if o != nil && !IsNil(o.Organization) {
		return true
	}

	return false
}

// SetOrganization gets a reference to the given OrganizationsOrganization and assigns it to the Organization field.
func (o *OrganizationsDescribeOrganizationResponse) SetOrganization(v OrganizationsOrganization) {
	o.Organization = &v
}

func (o OrganizationsDescribeOrganizationResponse) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o OrganizationsDescribeOrganizationResponse) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.Organization) {
		toSerialize["organization"] = o.Organization
	}
	return toSerialize, nil
}

type NullableOrganizationsDescribeOrganizationResponse struct {
	value *OrganizationsDescribeOrganizationResponse
	isSet bool
}

func (v NullableOrganizationsDescribeOrganizationResponse) Get() *OrganizationsDescribeOrganizationResponse {
	return v.value
}

func (v *NullableOrganizationsDescribeOrganizationResponse) Set(val *OrganizationsDescribeOrganizationResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableOrganizationsDescribeOrganizationResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableOrganizationsDescribeOrganizationResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableOrganizationsDescribeOrganizationResponse(val *OrganizationsDescribeOrganizationResponse) *NullableOrganizationsDescribeOrganizationResponse {
	return &NullableOrganizationsDescribeOrganizationResponse{value: val, isSet: true}
}

func (v NullableOrganizationsDescribeOrganizationResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableOrganizationsDescribeOrganizationResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


