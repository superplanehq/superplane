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

// checks if the SuperplaneUpdateStageBody type satisfies the MappedNullable interface at compile time
var _ MappedNullable = &SuperplaneUpdateStageBody{}

// SuperplaneUpdateStageBody struct for SuperplaneUpdateStageBody
type SuperplaneUpdateStageBody struct {
	RequesterId *string `json:"requesterId,omitempty"`
	Connections []SuperplaneConnection `json:"connections,omitempty"`
	Conditions []SuperplaneCondition `json:"conditions,omitempty"`
	Executor *SuperplaneExecutorSpec `json:"executor,omitempty"`
	Inputs []SuperplaneInputDefinition `json:"inputs,omitempty"`
	InputMappings []SuperplaneInputMapping `json:"inputMappings,omitempty"`
	Outputs []SuperplaneOutputDefinition `json:"outputs,omitempty"`
	Secrets []SuperplaneValueDefinition `json:"secrets,omitempty"`
}

// NewSuperplaneUpdateStageBody instantiates a new SuperplaneUpdateStageBody object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewSuperplaneUpdateStageBody() *SuperplaneUpdateStageBody {
	this := SuperplaneUpdateStageBody{}
	return &this
}

// NewSuperplaneUpdateStageBodyWithDefaults instantiates a new SuperplaneUpdateStageBody object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewSuperplaneUpdateStageBodyWithDefaults() *SuperplaneUpdateStageBody {
	this := SuperplaneUpdateStageBody{}
	return &this
}

// GetRequesterId returns the RequesterId field value if set, zero value otherwise.
func (o *SuperplaneUpdateStageBody) GetRequesterId() string {
	if o == nil || IsNil(o.RequesterId) {
		var ret string
		return ret
	}
	return *o.RequesterId
}

// GetRequesterIdOk returns a tuple with the RequesterId field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneUpdateStageBody) GetRequesterIdOk() (*string, bool) {
	if o == nil || IsNil(o.RequesterId) {
		return nil, false
	}
	return o.RequesterId, true
}

// HasRequesterId returns a boolean if a field has been set.
func (o *SuperplaneUpdateStageBody) HasRequesterId() bool {
	if o != nil && !IsNil(o.RequesterId) {
		return true
	}

	return false
}

// SetRequesterId gets a reference to the given string and assigns it to the RequesterId field.
func (o *SuperplaneUpdateStageBody) SetRequesterId(v string) {
	o.RequesterId = &v
}

// GetConnections returns the Connections field value if set, zero value otherwise.
func (o *SuperplaneUpdateStageBody) GetConnections() []SuperplaneConnection {
	if o == nil || IsNil(o.Connections) {
		var ret []SuperplaneConnection
		return ret
	}
	return o.Connections
}

// GetConnectionsOk returns a tuple with the Connections field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneUpdateStageBody) GetConnectionsOk() ([]SuperplaneConnection, bool) {
	if o == nil || IsNil(o.Connections) {
		return nil, false
	}
	return o.Connections, true
}

// HasConnections returns a boolean if a field has been set.
func (o *SuperplaneUpdateStageBody) HasConnections() bool {
	if o != nil && !IsNil(o.Connections) {
		return true
	}

	return false
}

// SetConnections gets a reference to the given []SuperplaneConnection and assigns it to the Connections field.
func (o *SuperplaneUpdateStageBody) SetConnections(v []SuperplaneConnection) {
	o.Connections = v
}

// GetConditions returns the Conditions field value if set, zero value otherwise.
func (o *SuperplaneUpdateStageBody) GetConditions() []SuperplaneCondition {
	if o == nil || IsNil(o.Conditions) {
		var ret []SuperplaneCondition
		return ret
	}
	return o.Conditions
}

// GetConditionsOk returns a tuple with the Conditions field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneUpdateStageBody) GetConditionsOk() ([]SuperplaneCondition, bool) {
	if o == nil || IsNil(o.Conditions) {
		return nil, false
	}
	return o.Conditions, true
}

// HasConditions returns a boolean if a field has been set.
func (o *SuperplaneUpdateStageBody) HasConditions() bool {
	if o != nil && !IsNil(o.Conditions) {
		return true
	}

	return false
}

// SetConditions gets a reference to the given []SuperplaneCondition and assigns it to the Conditions field.
func (o *SuperplaneUpdateStageBody) SetConditions(v []SuperplaneCondition) {
	o.Conditions = v
}

// GetExecutor returns the Executor field value if set, zero value otherwise.
func (o *SuperplaneUpdateStageBody) GetExecutor() SuperplaneExecutorSpec {
	if o == nil || IsNil(o.Executor) {
		var ret SuperplaneExecutorSpec
		return ret
	}
	return *o.Executor
}

// GetExecutorOk returns a tuple with the Executor field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneUpdateStageBody) GetExecutorOk() (*SuperplaneExecutorSpec, bool) {
	if o == nil || IsNil(o.Executor) {
		return nil, false
	}
	return o.Executor, true
}

// HasExecutor returns a boolean if a field has been set.
func (o *SuperplaneUpdateStageBody) HasExecutor() bool {
	if o != nil && !IsNil(o.Executor) {
		return true
	}

	return false
}

// SetExecutor gets a reference to the given SuperplaneExecutorSpec and assigns it to the Executor field.
func (o *SuperplaneUpdateStageBody) SetExecutor(v SuperplaneExecutorSpec) {
	o.Executor = &v
}

// GetInputs returns the Inputs field value if set, zero value otherwise.
func (o *SuperplaneUpdateStageBody) GetInputs() []SuperplaneInputDefinition {
	if o == nil || IsNil(o.Inputs) {
		var ret []SuperplaneInputDefinition
		return ret
	}
	return o.Inputs
}

// GetInputsOk returns a tuple with the Inputs field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneUpdateStageBody) GetInputsOk() ([]SuperplaneInputDefinition, bool) {
	if o == nil || IsNil(o.Inputs) {
		return nil, false
	}
	return o.Inputs, true
}

// HasInputs returns a boolean if a field has been set.
func (o *SuperplaneUpdateStageBody) HasInputs() bool {
	if o != nil && !IsNil(o.Inputs) {
		return true
	}

	return false
}

// SetInputs gets a reference to the given []SuperplaneInputDefinition and assigns it to the Inputs field.
func (o *SuperplaneUpdateStageBody) SetInputs(v []SuperplaneInputDefinition) {
	o.Inputs = v
}

// GetInputMappings returns the InputMappings field value if set, zero value otherwise.
func (o *SuperplaneUpdateStageBody) GetInputMappings() []SuperplaneInputMapping {
	if o == nil || IsNil(o.InputMappings) {
		var ret []SuperplaneInputMapping
		return ret
	}
	return o.InputMappings
}

// GetInputMappingsOk returns a tuple with the InputMappings field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneUpdateStageBody) GetInputMappingsOk() ([]SuperplaneInputMapping, bool) {
	if o == nil || IsNil(o.InputMappings) {
		return nil, false
	}
	return o.InputMappings, true
}

// HasInputMappings returns a boolean if a field has been set.
func (o *SuperplaneUpdateStageBody) HasInputMappings() bool {
	if o != nil && !IsNil(o.InputMappings) {
		return true
	}

	return false
}

// SetInputMappings gets a reference to the given []SuperplaneInputMapping and assigns it to the InputMappings field.
func (o *SuperplaneUpdateStageBody) SetInputMappings(v []SuperplaneInputMapping) {
	o.InputMappings = v
}

// GetOutputs returns the Outputs field value if set, zero value otherwise.
func (o *SuperplaneUpdateStageBody) GetOutputs() []SuperplaneOutputDefinition {
	if o == nil || IsNil(o.Outputs) {
		var ret []SuperplaneOutputDefinition
		return ret
	}
	return o.Outputs
}

// GetOutputsOk returns a tuple with the Outputs field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneUpdateStageBody) GetOutputsOk() ([]SuperplaneOutputDefinition, bool) {
	if o == nil || IsNil(o.Outputs) {
		return nil, false
	}
	return o.Outputs, true
}

// HasOutputs returns a boolean if a field has been set.
func (o *SuperplaneUpdateStageBody) HasOutputs() bool {
	if o != nil && !IsNil(o.Outputs) {
		return true
	}

	return false
}

// SetOutputs gets a reference to the given []SuperplaneOutputDefinition and assigns it to the Outputs field.
func (o *SuperplaneUpdateStageBody) SetOutputs(v []SuperplaneOutputDefinition) {
	o.Outputs = v
}

// GetSecrets returns the Secrets field value if set, zero value otherwise.
func (o *SuperplaneUpdateStageBody) GetSecrets() []SuperplaneValueDefinition {
	if o == nil || IsNil(o.Secrets) {
		var ret []SuperplaneValueDefinition
		return ret
	}
	return o.Secrets
}

// GetSecretsOk returns a tuple with the Secrets field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *SuperplaneUpdateStageBody) GetSecretsOk() ([]SuperplaneValueDefinition, bool) {
	if o == nil || IsNil(o.Secrets) {
		return nil, false
	}
	return o.Secrets, true
}

// HasSecrets returns a boolean if a field has been set.
func (o *SuperplaneUpdateStageBody) HasSecrets() bool {
	if o != nil && !IsNil(o.Secrets) {
		return true
	}

	return false
}

// SetSecrets gets a reference to the given []SuperplaneValueDefinition and assigns it to the Secrets field.
func (o *SuperplaneUpdateStageBody) SetSecrets(v []SuperplaneValueDefinition) {
	o.Secrets = v
}

func (o SuperplaneUpdateStageBody) MarshalJSON() ([]byte, error) {
	toSerialize,err := o.ToMap()
	if err != nil {
		return []byte{}, err
	}
	return json.Marshal(toSerialize)
}

func (o SuperplaneUpdateStageBody) ToMap() (map[string]interface{}, error) {
	toSerialize := map[string]interface{}{}
	if !IsNil(o.RequesterId) {
		toSerialize["requesterId"] = o.RequesterId
	}
	if !IsNil(o.Connections) {
		toSerialize["connections"] = o.Connections
	}
	if !IsNil(o.Conditions) {
		toSerialize["conditions"] = o.Conditions
	}
	if !IsNil(o.Executor) {
		toSerialize["executor"] = o.Executor
	}
	if !IsNil(o.Inputs) {
		toSerialize["inputs"] = o.Inputs
	}
	if !IsNil(o.InputMappings) {
		toSerialize["inputMappings"] = o.InputMappings
	}
	if !IsNil(o.Outputs) {
		toSerialize["outputs"] = o.Outputs
	}
	if !IsNil(o.Secrets) {
		toSerialize["secrets"] = o.Secrets
	}
	return toSerialize, nil
}

type NullableSuperplaneUpdateStageBody struct {
	value *SuperplaneUpdateStageBody
	isSet bool
}

func (v NullableSuperplaneUpdateStageBody) Get() *SuperplaneUpdateStageBody {
	return v.value
}

func (v *NullableSuperplaneUpdateStageBody) Set(val *SuperplaneUpdateStageBody) {
	v.value = val
	v.isSet = true
}

func (v NullableSuperplaneUpdateStageBody) IsSet() bool {
	return v.isSet
}

func (v *NullableSuperplaneUpdateStageBody) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableSuperplaneUpdateStageBody(val *SuperplaneUpdateStageBody) *NullableSuperplaneUpdateStageBody {
	return &NullableSuperplaneUpdateStageBody{value: val, isSet: true}
}

func (v NullableSuperplaneUpdateStageBody) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableSuperplaneUpdateStageBody) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


