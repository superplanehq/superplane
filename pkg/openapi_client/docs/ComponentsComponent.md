# ComponentsComponent

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** |  | [optional] 
**Label** | Pointer to **string** |  | [optional] 
**Description** | Pointer to **string** |  | [optional] 
**Configuration** | Pointer to [**[]ConfigurationField**](ConfigurationField.md) |  | [optional] 
**OutputChannels** | Pointer to [**[]SuperplaneComponentsOutputChannel**](SuperplaneComponentsOutputChannel.md) |  | [optional] 
**Icon** | Pointer to **string** |  | [optional] 
**Color** | Pointer to **string** |  | [optional] 
**ExampleOutput** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewComponentsComponent

`func NewComponentsComponent() *ComponentsComponent`

NewComponentsComponent instantiates a new ComponentsComponent object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewComponentsComponentWithDefaults

`func NewComponentsComponentWithDefaults() *ComponentsComponent`

NewComponentsComponentWithDefaults instantiates a new ComponentsComponent object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *ComponentsComponent) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *ComponentsComponent) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *ComponentsComponent) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *ComponentsComponent) HasName() bool`

HasName returns a boolean if a field has been set.

### GetLabel

`func (o *ComponentsComponent) GetLabel() string`

GetLabel returns the Label field if non-nil, zero value otherwise.

### GetLabelOk

`func (o *ComponentsComponent) GetLabelOk() (*string, bool)`

GetLabelOk returns a tuple with the Label field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabel

`func (o *ComponentsComponent) SetLabel(v string)`

SetLabel sets Label field to given value.

### HasLabel

`func (o *ComponentsComponent) HasLabel() bool`

HasLabel returns a boolean if a field has been set.

### GetDescription

`func (o *ComponentsComponent) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *ComponentsComponent) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *ComponentsComponent) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *ComponentsComponent) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetConfiguration

`func (o *ComponentsComponent) GetConfiguration() []ConfigurationField`

GetConfiguration returns the Configuration field if non-nil, zero value otherwise.

### GetConfigurationOk

`func (o *ComponentsComponent) GetConfigurationOk() (*[]ConfigurationField, bool)`

GetConfigurationOk returns a tuple with the Configuration field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfiguration

`func (o *ComponentsComponent) SetConfiguration(v []ConfigurationField)`

SetConfiguration sets Configuration field to given value.

### HasConfiguration

`func (o *ComponentsComponent) HasConfiguration() bool`

HasConfiguration returns a boolean if a field has been set.

### GetOutputChannels

`func (o *ComponentsComponent) GetOutputChannels() []SuperplaneComponentsOutputChannel`

GetOutputChannels returns the OutputChannels field if non-nil, zero value otherwise.

### GetOutputChannelsOk

`func (o *ComponentsComponent) GetOutputChannelsOk() (*[]SuperplaneComponentsOutputChannel, bool)`

GetOutputChannelsOk returns a tuple with the OutputChannels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOutputChannels

`func (o *ComponentsComponent) SetOutputChannels(v []SuperplaneComponentsOutputChannel)`

SetOutputChannels sets OutputChannels field to given value.

### HasOutputChannels

`func (o *ComponentsComponent) HasOutputChannels() bool`

HasOutputChannels returns a boolean if a field has been set.

### GetIcon

`func (o *ComponentsComponent) GetIcon() string`

GetIcon returns the Icon field if non-nil, zero value otherwise.

### GetIconOk

`func (o *ComponentsComponent) GetIconOk() (*string, bool)`

GetIconOk returns a tuple with the Icon field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIcon

`func (o *ComponentsComponent) SetIcon(v string)`

SetIcon sets Icon field to given value.

### HasIcon

`func (o *ComponentsComponent) HasIcon() bool`

HasIcon returns a boolean if a field has been set.

### GetColor

`func (o *ComponentsComponent) GetColor() string`

GetColor returns the Color field if non-nil, zero value otherwise.

### GetColorOk

`func (o *ComponentsComponent) GetColorOk() (*string, bool)`

GetColorOk returns a tuple with the Color field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetColor

`func (o *ComponentsComponent) SetColor(v string)`

SetColor sets Color field to given value.

### HasColor

`func (o *ComponentsComponent) HasColor() bool`

HasColor returns a boolean if a field has been set.

### GetExampleOutput

`func (o *ComponentsComponent) GetExampleOutput() map[string]interface{}`

GetExampleOutput returns the ExampleOutput field if non-nil, zero value otherwise.

### GetExampleOutputOk

`func (o *ComponentsComponent) GetExampleOutputOk() (*map[string]interface{}, bool)`

GetExampleOutputOk returns a tuple with the ExampleOutput field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExampleOutput

`func (o *ComponentsComponent) SetExampleOutput(v map[string]interface{})`

SetExampleOutput sets ExampleOutput field to given value.

### HasExampleOutput

`func (o *ComponentsComponent) HasExampleOutput() bool`

HasExampleOutput returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


