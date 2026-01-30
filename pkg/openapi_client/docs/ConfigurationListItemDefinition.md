# ConfigurationListItemDefinition

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Type** | Pointer to **string** |  | [optional] 
**Schema** | Pointer to [**[]ConfigurationField**](ConfigurationField.md) |  | [optional] 

## Methods

### NewConfigurationListItemDefinition

`func NewConfigurationListItemDefinition() *ConfigurationListItemDefinition`

NewConfigurationListItemDefinition instantiates a new ConfigurationListItemDefinition object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewConfigurationListItemDefinitionWithDefaults

`func NewConfigurationListItemDefinitionWithDefaults() *ConfigurationListItemDefinition`

NewConfigurationListItemDefinitionWithDefaults instantiates a new ConfigurationListItemDefinition object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetType

`func (o *ConfigurationListItemDefinition) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *ConfigurationListItemDefinition) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *ConfigurationListItemDefinition) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *ConfigurationListItemDefinition) HasType() bool`

HasType returns a boolean if a field has been set.

### GetSchema

`func (o *ConfigurationListItemDefinition) GetSchema() []ConfigurationField`

GetSchema returns the Schema field if non-nil, zero value otherwise.

### GetSchemaOk

`func (o *ConfigurationListItemDefinition) GetSchemaOk() (*[]ConfigurationField, bool)`

GetSchemaOk returns a tuple with the Schema field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSchema

`func (o *ConfigurationListItemDefinition) SetSchema(v []ConfigurationField)`

SetSchema sets Schema field to given value.

### HasSchema

`func (o *ConfigurationListItemDefinition) HasSchema() bool`

HasSchema returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


