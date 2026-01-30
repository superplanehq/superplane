# ConfigurationField

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** |  | [optional] 
**Type** | Pointer to **string** |  | [optional] 
**Description** | Pointer to **string** |  | [optional] 
**Required** | Pointer to **bool** |  | [optional] 
**DefaultValue** | Pointer to **string** |  | [optional] 
**Label** | Pointer to **string** |  | [optional] 
**VisibilityConditions** | Pointer to [**[]ConfigurationVisibilityCondition**](ConfigurationVisibilityCondition.md) |  | [optional] 
**TypeOptions** | Pointer to [**ConfigurationTypeOptions**](ConfigurationTypeOptions.md) |  | [optional] 
**RequiredConditions** | Pointer to [**[]ConfigurationRequiredCondition**](ConfigurationRequiredCondition.md) |  | [optional] 
**ValidationRules** | Pointer to [**[]ConfigurationValidationRule**](ConfigurationValidationRule.md) |  | [optional] 
**Placeholder** | Pointer to **string** |  | [optional] 
**Sensitive** | Pointer to **bool** |  | [optional] 
**Togglable** | Pointer to **bool** |  | [optional] 
**DisallowExpression** | Pointer to **bool** |  | [optional] 

## Methods

### NewConfigurationField

`func NewConfigurationField() *ConfigurationField`

NewConfigurationField instantiates a new ConfigurationField object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewConfigurationFieldWithDefaults

`func NewConfigurationFieldWithDefaults() *ConfigurationField`

NewConfigurationFieldWithDefaults instantiates a new ConfigurationField object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *ConfigurationField) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *ConfigurationField) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *ConfigurationField) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *ConfigurationField) HasName() bool`

HasName returns a boolean if a field has been set.

### GetType

`func (o *ConfigurationField) GetType() string`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *ConfigurationField) GetTypeOk() (*string, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *ConfigurationField) SetType(v string)`

SetType sets Type field to given value.

### HasType

`func (o *ConfigurationField) HasType() bool`

HasType returns a boolean if a field has been set.

### GetDescription

`func (o *ConfigurationField) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *ConfigurationField) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *ConfigurationField) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *ConfigurationField) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetRequired

`func (o *ConfigurationField) GetRequired() bool`

GetRequired returns the Required field if non-nil, zero value otherwise.

### GetRequiredOk

`func (o *ConfigurationField) GetRequiredOk() (*bool, bool)`

GetRequiredOk returns a tuple with the Required field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRequired

`func (o *ConfigurationField) SetRequired(v bool)`

SetRequired sets Required field to given value.

### HasRequired

`func (o *ConfigurationField) HasRequired() bool`

HasRequired returns a boolean if a field has been set.

### GetDefaultValue

`func (o *ConfigurationField) GetDefaultValue() string`

GetDefaultValue returns the DefaultValue field if non-nil, zero value otherwise.

### GetDefaultValueOk

`func (o *ConfigurationField) GetDefaultValueOk() (*string, bool)`

GetDefaultValueOk returns a tuple with the DefaultValue field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDefaultValue

`func (o *ConfigurationField) SetDefaultValue(v string)`

SetDefaultValue sets DefaultValue field to given value.

### HasDefaultValue

`func (o *ConfigurationField) HasDefaultValue() bool`

HasDefaultValue returns a boolean if a field has been set.

### GetLabel

`func (o *ConfigurationField) GetLabel() string`

GetLabel returns the Label field if non-nil, zero value otherwise.

### GetLabelOk

`func (o *ConfigurationField) GetLabelOk() (*string, bool)`

GetLabelOk returns a tuple with the Label field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLabel

`func (o *ConfigurationField) SetLabel(v string)`

SetLabel sets Label field to given value.

### HasLabel

`func (o *ConfigurationField) HasLabel() bool`

HasLabel returns a boolean if a field has been set.

### GetVisibilityConditions

`func (o *ConfigurationField) GetVisibilityConditions() []ConfigurationVisibilityCondition`

GetVisibilityConditions returns the VisibilityConditions field if non-nil, zero value otherwise.

### GetVisibilityConditionsOk

`func (o *ConfigurationField) GetVisibilityConditionsOk() (*[]ConfigurationVisibilityCondition, bool)`

GetVisibilityConditionsOk returns a tuple with the VisibilityConditions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVisibilityConditions

`func (o *ConfigurationField) SetVisibilityConditions(v []ConfigurationVisibilityCondition)`

SetVisibilityConditions sets VisibilityConditions field to given value.

### HasVisibilityConditions

`func (o *ConfigurationField) HasVisibilityConditions() bool`

HasVisibilityConditions returns a boolean if a field has been set.

### GetTypeOptions

`func (o *ConfigurationField) GetTypeOptions() ConfigurationTypeOptions`

GetTypeOptions returns the TypeOptions field if non-nil, zero value otherwise.

### GetTypeOptionsOk

`func (o *ConfigurationField) GetTypeOptionsOk() (*ConfigurationTypeOptions, bool)`

GetTypeOptionsOk returns a tuple with the TypeOptions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTypeOptions

`func (o *ConfigurationField) SetTypeOptions(v ConfigurationTypeOptions)`

SetTypeOptions sets TypeOptions field to given value.

### HasTypeOptions

`func (o *ConfigurationField) HasTypeOptions() bool`

HasTypeOptions returns a boolean if a field has been set.

### GetRequiredConditions

`func (o *ConfigurationField) GetRequiredConditions() []ConfigurationRequiredCondition`

GetRequiredConditions returns the RequiredConditions field if non-nil, zero value otherwise.

### GetRequiredConditionsOk

`func (o *ConfigurationField) GetRequiredConditionsOk() (*[]ConfigurationRequiredCondition, bool)`

GetRequiredConditionsOk returns a tuple with the RequiredConditions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRequiredConditions

`func (o *ConfigurationField) SetRequiredConditions(v []ConfigurationRequiredCondition)`

SetRequiredConditions sets RequiredConditions field to given value.

### HasRequiredConditions

`func (o *ConfigurationField) HasRequiredConditions() bool`

HasRequiredConditions returns a boolean if a field has been set.

### GetValidationRules

`func (o *ConfigurationField) GetValidationRules() []ConfigurationValidationRule`

GetValidationRules returns the ValidationRules field if non-nil, zero value otherwise.

### GetValidationRulesOk

`func (o *ConfigurationField) GetValidationRulesOk() (*[]ConfigurationValidationRule, bool)`

GetValidationRulesOk returns a tuple with the ValidationRules field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValidationRules

`func (o *ConfigurationField) SetValidationRules(v []ConfigurationValidationRule)`

SetValidationRules sets ValidationRules field to given value.

### HasValidationRules

`func (o *ConfigurationField) HasValidationRules() bool`

HasValidationRules returns a boolean if a field has been set.

### GetPlaceholder

`func (o *ConfigurationField) GetPlaceholder() string`

GetPlaceholder returns the Placeholder field if non-nil, zero value otherwise.

### GetPlaceholderOk

`func (o *ConfigurationField) GetPlaceholderOk() (*string, bool)`

GetPlaceholderOk returns a tuple with the Placeholder field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPlaceholder

`func (o *ConfigurationField) SetPlaceholder(v string)`

SetPlaceholder sets Placeholder field to given value.

### HasPlaceholder

`func (o *ConfigurationField) HasPlaceholder() bool`

HasPlaceholder returns a boolean if a field has been set.

### GetSensitive

`func (o *ConfigurationField) GetSensitive() bool`

GetSensitive returns the Sensitive field if non-nil, zero value otherwise.

### GetSensitiveOk

`func (o *ConfigurationField) GetSensitiveOk() (*bool, bool)`

GetSensitiveOk returns a tuple with the Sensitive field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSensitive

`func (o *ConfigurationField) SetSensitive(v bool)`

SetSensitive sets Sensitive field to given value.

### HasSensitive

`func (o *ConfigurationField) HasSensitive() bool`

HasSensitive returns a boolean if a field has been set.

### GetTogglable

`func (o *ConfigurationField) GetTogglable() bool`

GetTogglable returns the Togglable field if non-nil, zero value otherwise.

### GetTogglableOk

`func (o *ConfigurationField) GetTogglableOk() (*bool, bool)`

GetTogglableOk returns a tuple with the Togglable field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTogglable

`func (o *ConfigurationField) SetTogglable(v bool)`

SetTogglable sets Togglable field to given value.

### HasTogglable

`func (o *ConfigurationField) HasTogglable() bool`

HasTogglable returns a boolean if a field has been set.

### GetDisallowExpression

`func (o *ConfigurationField) GetDisallowExpression() bool`

GetDisallowExpression returns the DisallowExpression field if non-nil, zero value otherwise.

### GetDisallowExpressionOk

`func (o *ConfigurationField) GetDisallowExpressionOk() (*bool, bool)`

GetDisallowExpressionOk returns a tuple with the DisallowExpression field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDisallowExpression

`func (o *ConfigurationField) SetDisallowExpression(v bool)`

SetDisallowExpression sets DisallowExpression field to given value.

### HasDisallowExpression

`func (o *ConfigurationField) HasDisallowExpression() bool`

HasDisallowExpression returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


