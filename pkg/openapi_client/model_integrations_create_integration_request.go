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

// checks if the IntegrationsCreateIntegrationRequest type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &IntegrationsCreateIntegrationRequest{}

// IntegrationsCreateIntegrationRequest struct for IntegrationsCreateIntegrationRequest
type IntegrationsCreateIntegrationRequest struct {
	DomainType *AuthorizationDomainType `json:"domainType,omitempty"`
	DomainId *string `json:"domainId,omitempty"`
	Integration *IntegrationsIntegration `json:"integration,omitempty"`
}

// NewIntegrationsCreateIntegrationRequest instantiates a new IntegrationsCreateIntegrationRequest object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewIntegrationsCreateIntegrationRequest() *IntegrationsCreateIntegrationRequest {
	this := IntegrationsCreateIntegrationRequest{}
	var domainType AuthorizationDomainType = AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_UNSPECIFIED
	this.DomainType = &domainType
	return &this
}

// NewIntegrationsCreateIntegrationRequestWithDefaults instantiates a new IntegrationsCreateIntegrationRequest object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewIntegrationsCreateIntegrationRequestWithDefaults() *IntegrationsCreateIntegrationRequest {
	this := IntegrationsCreateIntegrationRequest{}
	var domainType AuthorizationDomainType = AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_UNSPECIFIED
	this.DomainType = &domainType
	return &this
}

// GetDomainType returns the DomainType field value if set, zero value otherwise.
func (o *IntegrationsCreateIntegrationRequest) GetDomainType() AuthorizationDomainType {
	if o == nil || IsNil(o.DomainType) {
		var ret AuthorizationDomainType
		return ret
	}
	return *o.DomainType
}

// GetDomainTypeOk returns a tuple with the DomainType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IntegrationsCreateIntegrationRequest) GetDomainTypeOk() (*AuthorizationDomainType, bool) {
	if o == nil || IsNil(o.DomainType) {
		return nil, false
	}
	return o.DomainType, true
}

// HasDomainType returns a boolean if a field has been set.
func (o *IntegrationsCreateIntegrationRequest) HasDomainType() bool {
	if o != nil && !IsNil(o.DomainType) {
		return true
	}

	return false
}

// SetDomainType gets a reference to the given AuthorizationDomainType and assigns it to the DomainType field.
func (o *IntegrationsCreateIntegrationRequest) SetDomainType(v AuthorizationDomainType) {
	o.DomainType = &v
}

// GetDomainId returns the DomainId field value if set, zero value otherwise.
func (o *IntegrationsCreateIntegrationRequest) GetDomainId() string {
	if o == nil || IsNil(o.DomainId) {
		var ret string
		return ret
	}
	return *o.DomainId
}

// GetDomainIdOk returns a tuple with the DomainId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IntegrationsCreateIntegrationRequest) GetDomainIdOk() (*string, bool) {
	if o == nil || IsNil(o.DomainId) {
		return nil, false
	}
	return o.DomainId, true
}

// HasDomainId returns a boolean if a field has been set.
func (o *IntegrationsCreateIntegrationRequest) HasDomainId() bool {
	if o != nil && !IsNil(o.DomainId) {
		return true
	}

	return false
}

// SetDomainId gets a reference to the given string and assigns it to the DomainId field.
func (o *IntegrationsCreateIntegrationRequest) SetDomainId(v string) {
	o.DomainId = &v
}

// GetIntegration returns the Integration field value if set, zero value otherwise.
func (o *IntegrationsCreateIntegrationRequest) GetIntegration() IntegrationsIntegration {
	if o == nil || IsNil(o.Integration) {
		var ret IntegrationsIntegration
		return ret
	}
	return *o.Integration
}

// GetIntegrationOk returns a tuple with the Integration field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *IntegrationsCreateIntegrationRequest) GetIntegrationOk() (*IntegrationsIntegration, bool) {
	if o == nil || IsNil(o.Integration) {
		return nil, false
	}
	return o.Integration, true
}

// HasIntegration returns a boolean if a field has been set.
func (o *IntegrationsCreateIntegrationRequest) HasIntegration() bool {
	if o != nil && !IsNil(o.Integration) {
		return true
	}

	return false
}

// SetIntegration gets a reference to the given IntegrationsIntegration and assigns it to the Integration field.
func (o *IntegrationsCreateIntegrationRequest) SetIntegration(v IntegrationsIntegration) {
	o.Integration = &v
}

func (o IntegrationsCreateIntegrationRequest) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o IntegrationsCreateIntegrationRequest) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.DomainType) {
		toSerialize["domainType"] = o.DomainType
	}
	if !IsNil(o.DomainId) {
		toSerialize["domainId"] = o.DomainId
	}
	if !IsNil(o.Integration) {
		toSerialize["integration"] = o.Integration
	}
	return toSerialize, nil
}

type NullableIntegrationsCreateIntegrationRequest struct {
	value *IntegrationsCreateIntegrationRequest
	isSet bool
}

func (v NullableIntegrationsCreateIntegrationRequest) Get() *IntegrationsCreateIntegrationRequest {
	return v.value
}

func (v *NullableIntegrationsCreateIntegrationRequest) Set(val *IntegrationsCreateIntegrationRequest) {
	v.value = val
	v.isSet = true
}

func (v NullableIntegrationsCreateIntegrationRequest) IsSet() bool {
	return v.isSet
}

func (v *NullableIntegrationsCreateIntegrationRequest) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableIntegrationsCreateIntegrationRequest(val *IntegrationsCreateIntegrationRequest) *NullableIntegrationsCreateIntegrationRequest {
	return &NullableIntegrationsCreateIntegrationRequest{value: val, isSet: true}
}

func (v NullableIntegrationsCreateIntegrationRequest) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableIntegrationsCreateIntegrationRequest) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


