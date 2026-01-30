# ConfigurationTypeOptions

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Number** | Pointer to [**ConfigurationNumberTypeOptions**](ConfigurationNumberTypeOptions.md) |  | [optional] 
**Select** | Pointer to [**ConfigurationSelectTypeOptions**](ConfigurationSelectTypeOptions.md) |  | [optional] 
**MultiSelect** | Pointer to [**ConfigurationMultiSelectTypeOptions**](ConfigurationMultiSelectTypeOptions.md) |  | [optional] 
**List** | Pointer to [**ConfigurationListTypeOptions**](ConfigurationListTypeOptions.md) |  | [optional] 
**Object** | Pointer to [**ConfigurationObjectTypeOptions**](ConfigurationObjectTypeOptions.md) |  | [optional] 
**Resource** | Pointer to [**ConfigurationResourceTypeOptions**](ConfigurationResourceTypeOptions.md) |  | [optional] 
**Time** | Pointer to [**ConfigurationTimeTypeOptions**](ConfigurationTimeTypeOptions.md) |  | [optional] 
**Date** | Pointer to [**ConfigurationDateTypeOptions**](ConfigurationDateTypeOptions.md) |  | [optional] 
**Datetime** | Pointer to [**ConfigurationDateTimeTypeOptions**](ConfigurationDateTimeTypeOptions.md) |  | [optional] 
**AnyPredicateList** | Pointer to [**ConfigurationAnyPredicateListTypeOptions**](ConfigurationAnyPredicateListTypeOptions.md) |  | [optional] 
**String** | Pointer to [**ConfigurationStringTypeOptions**](ConfigurationStringTypeOptions.md) |  | [optional] 
**Expression** | Pointer to [**ConfigurationExpressionTypeOptions**](ConfigurationExpressionTypeOptions.md) |  | [optional] 
**Text** | Pointer to [**ConfigurationTextTypeOptions**](ConfigurationTextTypeOptions.md) |  | [optional] 

## Methods

### NewConfigurationTypeOptions

`func NewConfigurationTypeOptions() *ConfigurationTypeOptions`

NewConfigurationTypeOptions instantiates a new ConfigurationTypeOptions object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewConfigurationTypeOptionsWithDefaults

`func NewConfigurationTypeOptionsWithDefaults() *ConfigurationTypeOptions`

NewConfigurationTypeOptionsWithDefaults instantiates a new ConfigurationTypeOptions object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetNumber

`func (o *ConfigurationTypeOptions) GetNumber() ConfigurationNumberTypeOptions`

GetNumber returns the Number field if non-nil, zero value otherwise.

### GetNumberOk

`func (o *ConfigurationTypeOptions) GetNumberOk() (*ConfigurationNumberTypeOptions, bool)`

GetNumberOk returns a tuple with the Number field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNumber

`func (o *ConfigurationTypeOptions) SetNumber(v ConfigurationNumberTypeOptions)`

SetNumber sets Number field to given value.

### HasNumber

`func (o *ConfigurationTypeOptions) HasNumber() bool`

HasNumber returns a boolean if a field has been set.

### GetSelect

`func (o *ConfigurationTypeOptions) GetSelect() ConfigurationSelectTypeOptions`

GetSelect returns the Select field if non-nil, zero value otherwise.

### GetSelectOk

`func (o *ConfigurationTypeOptions) GetSelectOk() (*ConfigurationSelectTypeOptions, bool)`

GetSelectOk returns a tuple with the Select field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSelect

`func (o *ConfigurationTypeOptions) SetSelect(v ConfigurationSelectTypeOptions)`

SetSelect sets Select field to given value.

### HasSelect

`func (o *ConfigurationTypeOptions) HasSelect() bool`

HasSelect returns a boolean if a field has been set.

### GetMultiSelect

`func (o *ConfigurationTypeOptions) GetMultiSelect() ConfigurationMultiSelectTypeOptions`

GetMultiSelect returns the MultiSelect field if non-nil, zero value otherwise.

### GetMultiSelectOk

`func (o *ConfigurationTypeOptions) GetMultiSelectOk() (*ConfigurationMultiSelectTypeOptions, bool)`

GetMultiSelectOk returns a tuple with the MultiSelect field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMultiSelect

`func (o *ConfigurationTypeOptions) SetMultiSelect(v ConfigurationMultiSelectTypeOptions)`

SetMultiSelect sets MultiSelect field to given value.

### HasMultiSelect

`func (o *ConfigurationTypeOptions) HasMultiSelect() bool`

HasMultiSelect returns a boolean if a field has been set.

### GetList

`func (o *ConfigurationTypeOptions) GetList() ConfigurationListTypeOptions`

GetList returns the List field if non-nil, zero value otherwise.

### GetListOk

`func (o *ConfigurationTypeOptions) GetListOk() (*ConfigurationListTypeOptions, bool)`

GetListOk returns a tuple with the List field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetList

`func (o *ConfigurationTypeOptions) SetList(v ConfigurationListTypeOptions)`

SetList sets List field to given value.

### HasList

`func (o *ConfigurationTypeOptions) HasList() bool`

HasList returns a boolean if a field has been set.

### GetObject

`func (o *ConfigurationTypeOptions) GetObject() ConfigurationObjectTypeOptions`

GetObject returns the Object field if non-nil, zero value otherwise.

### GetObjectOk

`func (o *ConfigurationTypeOptions) GetObjectOk() (*ConfigurationObjectTypeOptions, bool)`

GetObjectOk returns a tuple with the Object field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetObject

`func (o *ConfigurationTypeOptions) SetObject(v ConfigurationObjectTypeOptions)`

SetObject sets Object field to given value.

### HasObject

`func (o *ConfigurationTypeOptions) HasObject() bool`

HasObject returns a boolean if a field has been set.

### GetResource

`func (o *ConfigurationTypeOptions) GetResource() ConfigurationResourceTypeOptions`

GetResource returns the Resource field if non-nil, zero value otherwise.

### GetResourceOk

`func (o *ConfigurationTypeOptions) GetResourceOk() (*ConfigurationResourceTypeOptions, bool)`

GetResourceOk returns a tuple with the Resource field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResource

`func (o *ConfigurationTypeOptions) SetResource(v ConfigurationResourceTypeOptions)`

SetResource sets Resource field to given value.

### HasResource

`func (o *ConfigurationTypeOptions) HasResource() bool`

HasResource returns a boolean if a field has been set.

### GetTime

`func (o *ConfigurationTypeOptions) GetTime() ConfigurationTimeTypeOptions`

GetTime returns the Time field if non-nil, zero value otherwise.

### GetTimeOk

`func (o *ConfigurationTypeOptions) GetTimeOk() (*ConfigurationTimeTypeOptions, bool)`

GetTimeOk returns a tuple with the Time field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTime

`func (o *ConfigurationTypeOptions) SetTime(v ConfigurationTimeTypeOptions)`

SetTime sets Time field to given value.

### HasTime

`func (o *ConfigurationTypeOptions) HasTime() bool`

HasTime returns a boolean if a field has been set.

### GetDate

`func (o *ConfigurationTypeOptions) GetDate() ConfigurationDateTypeOptions`

GetDate returns the Date field if non-nil, zero value otherwise.

### GetDateOk

`func (o *ConfigurationTypeOptions) GetDateOk() (*ConfigurationDateTypeOptions, bool)`

GetDateOk returns a tuple with the Date field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDate

`func (o *ConfigurationTypeOptions) SetDate(v ConfigurationDateTypeOptions)`

SetDate sets Date field to given value.

### HasDate

`func (o *ConfigurationTypeOptions) HasDate() bool`

HasDate returns a boolean if a field has been set.

### GetDatetime

`func (o *ConfigurationTypeOptions) GetDatetime() ConfigurationDateTimeTypeOptions`

GetDatetime returns the Datetime field if non-nil, zero value otherwise.

### GetDatetimeOk

`func (o *ConfigurationTypeOptions) GetDatetimeOk() (*ConfigurationDateTimeTypeOptions, bool)`

GetDatetimeOk returns a tuple with the Datetime field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDatetime

`func (o *ConfigurationTypeOptions) SetDatetime(v ConfigurationDateTimeTypeOptions)`

SetDatetime sets Datetime field to given value.

### HasDatetime

`func (o *ConfigurationTypeOptions) HasDatetime() bool`

HasDatetime returns a boolean if a field has been set.

### GetAnyPredicateList

`func (o *ConfigurationTypeOptions) GetAnyPredicateList() ConfigurationAnyPredicateListTypeOptions`

GetAnyPredicateList returns the AnyPredicateList field if non-nil, zero value otherwise.

### GetAnyPredicateListOk

`func (o *ConfigurationTypeOptions) GetAnyPredicateListOk() (*ConfigurationAnyPredicateListTypeOptions, bool)`

GetAnyPredicateListOk returns a tuple with the AnyPredicateList field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAnyPredicateList

`func (o *ConfigurationTypeOptions) SetAnyPredicateList(v ConfigurationAnyPredicateListTypeOptions)`

SetAnyPredicateList sets AnyPredicateList field to given value.

### HasAnyPredicateList

`func (o *ConfigurationTypeOptions) HasAnyPredicateList() bool`

HasAnyPredicateList returns a boolean if a field has been set.

### GetString

`func (o *ConfigurationTypeOptions) GetString() ConfigurationStringTypeOptions`

GetString returns the String field if non-nil, zero value otherwise.

### GetStringOk

`func (o *ConfigurationTypeOptions) GetStringOk() (*ConfigurationStringTypeOptions, bool)`

GetStringOk returns a tuple with the String field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetString

`func (o *ConfigurationTypeOptions) SetString(v ConfigurationStringTypeOptions)`

SetString sets String field to given value.

### HasString

`func (o *ConfigurationTypeOptions) HasString() bool`

HasString returns a boolean if a field has been set.

### GetExpression

`func (o *ConfigurationTypeOptions) GetExpression() ConfigurationExpressionTypeOptions`

GetExpression returns the Expression field if non-nil, zero value otherwise.

### GetExpressionOk

`func (o *ConfigurationTypeOptions) GetExpressionOk() (*ConfigurationExpressionTypeOptions, bool)`

GetExpressionOk returns a tuple with the Expression field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpression

`func (o *ConfigurationTypeOptions) SetExpression(v ConfigurationExpressionTypeOptions)`

SetExpression sets Expression field to given value.

### HasExpression

`func (o *ConfigurationTypeOptions) HasExpression() bool`

HasExpression returns a boolean if a field has been set.

### GetText

`func (o *ConfigurationTypeOptions) GetText() ConfigurationTextTypeOptions`

GetText returns the Text field if non-nil, zero value otherwise.

### GetTextOk

`func (o *ConfigurationTypeOptions) GetTextOk() (*ConfigurationTextTypeOptions, bool)`

GetTextOk returns a tuple with the Text field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetText

`func (o *ConfigurationTypeOptions) SetText(v ConfigurationTextTypeOptions)`

SetText sets Text field to given value.

### HasText

`func (o *ConfigurationTypeOptions) HasText() bool`

HasText returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


