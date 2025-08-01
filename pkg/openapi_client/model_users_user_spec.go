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

// checks if the UsersUserSpec type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &UsersUserSpec{}

// UsersUserSpec struct for UsersUserSpec
type UsersUserSpec struct {
	DisplayName *string `json:"displayName,omitempty"`
	AvatarUrl *string `json:"avatarUrl,omitempty"`
	AccountProviders []UsersAccountProvider `json:"accountProviders,omitempty"`
}

// NewUsersUserSpec instantiates a new UsersUserSpec object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewUsersUserSpec() *UsersUserSpec {
	this := UsersUserSpec{}
	return &this
}

// NewUsersUserSpecWithDefaults instantiates a new UsersUserSpec object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewUsersUserSpecWithDefaults() *UsersUserSpec {
	this := UsersUserSpec{}
	return &this
}

// GetDisplayName returns the DisplayName field value if set, zero value otherwise.
func (o *UsersUserSpec) GetDisplayName() string {
	if o == nil || IsNil(o.DisplayName) {
		var ret string
		return ret
	}
	return *o.DisplayName
}

// GetDisplayNameOk returns a tuple with the DisplayName field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsersUserSpec) GetDisplayNameOk() (*string, bool) {
	if o == nil || IsNil(o.DisplayName) {
		return nil, false
	}
	return o.DisplayName, true
}

// HasDisplayName returns a boolean if a field has been set.
func (o *UsersUserSpec) HasDisplayName() bool {
	if o != nil && !IsNil(o.DisplayName) {
		return true
	}

	return false
}

// SetDisplayName gets a reference to the given string and assigns it to the DisplayName field.
func (o *UsersUserSpec) SetDisplayName(v string) {
	o.DisplayName = &v
}

// GetAvatarUrl returns the AvatarUrl field value if set, zero value otherwise.
func (o *UsersUserSpec) GetAvatarUrl() string {
	if o == nil || IsNil(o.AvatarUrl) {
		var ret string
		return ret
	}
	return *o.AvatarUrl
}

// GetAvatarUrlOk returns a tuple with the AvatarUrl field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsersUserSpec) GetAvatarUrlOk() (*string, bool) {
	if o == nil || IsNil(o.AvatarUrl) {
		return nil, false
	}
	return o.AvatarUrl, true
}

// HasAvatarUrl returns a boolean if a field has been set.
func (o *UsersUserSpec) HasAvatarUrl() bool {
	if o != nil && !IsNil(o.AvatarUrl) {
		return true
	}

	return false
}

// SetAvatarUrl gets a reference to the given string and assigns it to the AvatarUrl field.
func (o *UsersUserSpec) SetAvatarUrl(v string) {
	o.AvatarUrl = &v
}

// GetAccountProviders returns the AccountProviders field value if set, zero value otherwise.
func (o *UsersUserSpec) GetAccountProviders() []UsersAccountProvider {
	if o == nil || IsNil(o.AccountProviders) {
		var ret []UsersAccountProvider
		return ret
	}
	return o.AccountProviders
}

// GetAccountProvidersOk returns a tuple with the AccountProviders field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UsersUserSpec) GetAccountProvidersOk() ([]UsersAccountProvider, bool) {
	if o == nil || IsNil(o.AccountProviders) {
		return nil, false
	}
	return o.AccountProviders, true
}

// HasAccountProviders returns a boolean if a field has been set.
func (o *UsersUserSpec) HasAccountProviders() bool {
	if o != nil && !IsNil(o.AccountProviders) {
		return true
	}

	return false
}

// SetAccountProviders gets a reference to the given []UsersAccountProvider and assigns it to the AccountProviders field.
func (o *UsersUserSpec) SetAccountProviders(v []UsersAccountProvider) {
	o.AccountProviders = v
}

func (o UsersUserSpec) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o UsersUserSpec) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.DisplayName) {
		toSerialize["displayName"] = o.DisplayName
	}
	if !IsNil(o.AvatarUrl) {
		toSerialize["avatarUrl"] = o.AvatarUrl
	}
	if !IsNil(o.AccountProviders) {
		toSerialize["accountProviders"] = o.AccountProviders
	}
	return toSerialize, nil
}

type NullableUsersUserSpec struct {
	value *UsersUserSpec
	isSet bool
}

func (v NullableUsersUserSpec) Get() *UsersUserSpec {
	return v.value
}

func (v *NullableUsersUserSpec) Set(val *UsersUserSpec) {
	v.value = val
	v.isSet = true
}

func (v NullableUsersUserSpec) IsSet() bool {
	return v.isSet
}

func (v *NullableUsersUserSpec) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableUsersUserSpec(val *UsersUserSpec) *NullableUsersUserSpec {
	return &NullableUsersUserSpec{value: val, isSet: true}
}

func (v NullableUsersUserSpec) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableUsersUserSpec) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


