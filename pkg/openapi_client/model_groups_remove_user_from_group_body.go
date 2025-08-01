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

// checks if the GroupsRemoveUserFromGroupBody type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &GroupsRemoveUserFromGroupBody{}

// GroupsRemoveUserFromGroupBody struct for GroupsRemoveUserFromGroupBody
type GroupsRemoveUserFromGroupBody struct {
	DomainType *AuthorizationDomainType `json:"domainType,omitempty"`
	DomainId *string `json:"domainId,omitempty"`
	UserId *string `json:"userId,omitempty"`
	UserEmail *string `json:"userEmail,omitempty"`
}

// NewGroupsRemoveUserFromGroupBody instantiates a new GroupsRemoveUserFromGroupBody object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewGroupsRemoveUserFromGroupBody() *GroupsRemoveUserFromGroupBody {
	this := GroupsRemoveUserFromGroupBody{}
	var domainType AuthorizationDomainType = AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_UNSPECIFIED
	this.DomainType = &domainType
	return &this
}

// NewGroupsRemoveUserFromGroupBodyWithDefaults instantiates a new GroupsRemoveUserFromGroupBody object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewGroupsRemoveUserFromGroupBodyWithDefaults() *GroupsRemoveUserFromGroupBody {
	this := GroupsRemoveUserFromGroupBody{}
	var domainType AuthorizationDomainType = AUTHORIZATIONDOMAINTYPE_DOMAIN_TYPE_UNSPECIFIED
	this.DomainType = &domainType
	return &this
}

// GetDomainType returns the DomainType field value if set, zero value otherwise.
func (o *GroupsRemoveUserFromGroupBody) GetDomainType() AuthorizationDomainType {
	if o == nil || IsNil(o.DomainType) {
		var ret AuthorizationDomainType
		return ret
	}
	return *o.DomainType
}

// GetDomainTypeOk returns a tuple with the DomainType field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GroupsRemoveUserFromGroupBody) GetDomainTypeOk() (*AuthorizationDomainType, bool) {
	if o == nil || IsNil(o.DomainType) {
		return nil, false
	}
	return o.DomainType, true
}

// HasDomainType returns a boolean if a field has been set.
func (o *GroupsRemoveUserFromGroupBody) HasDomainType() bool {
	if o != nil && !IsNil(o.DomainType) {
		return true
	}

	return false
}

// SetDomainType gets a reference to the given AuthorizationDomainType and assigns it to the DomainType field.
func (o *GroupsRemoveUserFromGroupBody) SetDomainType(v AuthorizationDomainType) {
	o.DomainType = &v
}

// GetDomainId returns the DomainId field value if set, zero value otherwise.
func (o *GroupsRemoveUserFromGroupBody) GetDomainId() string {
	if o == nil || IsNil(o.DomainId) {
		var ret string
		return ret
	}
	return *o.DomainId
}

// GetDomainIdOk returns a tuple with the DomainId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GroupsRemoveUserFromGroupBody) GetDomainIdOk() (*string, bool) {
	if o == nil || IsNil(o.DomainId) {
		return nil, false
	}
	return o.DomainId, true
}

// HasDomainId returns a boolean if a field has been set.
func (o *GroupsRemoveUserFromGroupBody) HasDomainId() bool {
	if o != nil && !IsNil(o.DomainId) {
		return true
	}

	return false
}

// SetDomainId gets a reference to the given string and assigns it to the DomainId field.
func (o *GroupsRemoveUserFromGroupBody) SetDomainId(v string) {
	o.DomainId = &v
}

// GetUserId returns the UserId field value if set, zero value otherwise.
func (o *GroupsRemoveUserFromGroupBody) GetUserId() string {
	if o == nil || IsNil(o.UserId) {
		var ret string
		return ret
	}
	return *o.UserId
}

// GetUserIdOk returns a tuple with the UserId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GroupsRemoveUserFromGroupBody) GetUserIdOk() (*string, bool) {
	if o == nil || IsNil(o.UserId) {
		return nil, false
	}
	return o.UserId, true
}

// HasUserId returns a boolean if a field has been set.
func (o *GroupsRemoveUserFromGroupBody) HasUserId() bool {
	if o != nil && !IsNil(o.UserId) {
		return true
	}

	return false
}

// SetUserId gets a reference to the given string and assigns it to the UserId field.
func (o *GroupsRemoveUserFromGroupBody) SetUserId(v string) {
	o.UserId = &v
}

// GetUserEmail returns the UserEmail field value if set, zero value otherwise.
func (o *GroupsRemoveUserFromGroupBody) GetUserEmail() string {
	if o == nil || IsNil(o.UserEmail) {
		var ret string
		return ret
	}
	return *o.UserEmail
}

// GetUserEmailOk returns a tuple with the UserEmail field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *GroupsRemoveUserFromGroupBody) GetUserEmailOk() (*string, bool) {
	if o == nil || IsNil(o.UserEmail) {
		return nil, false
	}
	return o.UserEmail, true
}

// HasUserEmail returns a boolean if a field has been set.
func (o *GroupsRemoveUserFromGroupBody) HasUserEmail() bool {
	if o != nil && !IsNil(o.UserEmail) {
		return true
	}

	return false
}

// SetUserEmail gets a reference to the given string and assigns it to the UserEmail field.
func (o *GroupsRemoveUserFromGroupBody) SetUserEmail(v string) {
	o.UserEmail = &v
}

func (o GroupsRemoveUserFromGroupBody) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o GroupsRemoveUserFromGroupBody) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.DomainType) {
		toSerialize["domainType"] = o.DomainType
	}
	if !IsNil(o.DomainId) {
		toSerialize["domainId"] = o.DomainId
	}
	if !IsNil(o.UserId) {
		toSerialize["userId"] = o.UserId
	}
	if !IsNil(o.UserEmail) {
		toSerialize["userEmail"] = o.UserEmail
	}
	return toSerialize, nil
}

type NullableGroupsRemoveUserFromGroupBody struct {
	value *GroupsRemoveUserFromGroupBody
	isSet bool
}

func (v NullableGroupsRemoveUserFromGroupBody) Get() *GroupsRemoveUserFromGroupBody {
	return v.value
}

func (v *NullableGroupsRemoveUserFromGroupBody) Set(val *GroupsRemoveUserFromGroupBody) {
	v.value = val
	v.isSet = true
}

func (v NullableGroupsRemoveUserFromGroupBody) IsSet() bool {
	return v.isSet
}

func (v *NullableGroupsRemoveUserFromGroupBody) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableGroupsRemoveUserFromGroupBody(val *GroupsRemoveUserFromGroupBody) *NullableGroupsRemoveUserFromGroupBody {
	return &NullableGroupsRemoveUserFromGroupBody{value: val, isSet: true}
}

func (v NullableGroupsRemoveUserFromGroupBody) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableGroupsRemoveUserFromGroupBody) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


