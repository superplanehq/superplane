# ComponentsNode

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**Name** | Pointer to **string** |  | [optional] 
**Type** | Pointer to [**ComponentsNodeType**](ComponentsNodeType.md) |  | [optional] [default to COMPONENTSNODETYPE_TYPE_COMPONENT]
**Configuration** | Pointer to **map[string]interface{}** |  | [optional] 
**Metadata** | Pointer to **map[string]interface{}** |  | [optional] 
**Position** | Pointer to [**ComponentsPosition**](ComponentsPosition.md) |  | [optional] 
**Component** | Pointer to [**NodeComponentRef**](NodeComponentRef.md) |  | [optional] 
**Blueprint** | Pointer to [**NodeBlueprintRef**](NodeBlueprintRef.md) |  | [optional] 
**Trigger** | Pointer to [**NodeTriggerRef**](NodeTriggerRef.md) |  | [optional] 
**Widget** | Pointer to [**NodeWidgetRef**](NodeWidgetRef.md) |  | [optional] 
**IsCollapsed** | Pointer to **bool** |  | [optional] 
**Integration** | Pointer to [**ComponentsIntegrationRef**](ComponentsIntegrationRef.md) |  | [optional] 
**ErrorMessage** | Pointer to **string** |  | [optional] 
**WarningMessage** | Pointer to **string** |  | [optional] 
**Paused** | Pointer to **bool** |  | [optional] 

## Methods

### NewComponentsNode

`func NewComponentsNode() *ComponentsNode`

NewComponentsNode instantiates a new ComponentsNode object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewComponentsNodeWithDefaults

`func NewComponentsNodeWithDefaults() *ComponentsNode`

NewComponentsNodeWithDefaults instantiates a new ComponentsNode object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *ComponentsNode) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *ComponentsNode) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *ComponentsNode) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *ComponentsNode) HasId() bool`

HasId returns a boolean if a field has been set.

### GetName

`func (o *ComponentsNode) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *ComponentsNode) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *ComponentsNode) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *ComponentsNode) HasName() bool`

HasName returns a boolean if a field has been set.

### GetType

`func (o *ComponentsNode) GetType() ComponentsNodeType`

GetType returns the Type field if non-nil, zero value otherwise.

### GetTypeOk

`func (o *ComponentsNode) GetTypeOk() (*ComponentsNodeType, bool)`

GetTypeOk returns a tuple with the Type field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetType

`func (o *ComponentsNode) SetType(v ComponentsNodeType)`

SetType sets Type field to given value.

### HasType

`func (o *ComponentsNode) HasType() bool`

HasType returns a boolean if a field has been set.

### GetConfiguration

`func (o *ComponentsNode) GetConfiguration() map[string]interface{}`

GetConfiguration returns the Configuration field if non-nil, zero value otherwise.

### GetConfigurationOk

`func (o *ComponentsNode) GetConfigurationOk() (*map[string]interface{}, bool)`

GetConfigurationOk returns a tuple with the Configuration field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfiguration

`func (o *ComponentsNode) SetConfiguration(v map[string]interface{})`

SetConfiguration sets Configuration field to given value.

### HasConfiguration

`func (o *ComponentsNode) HasConfiguration() bool`

HasConfiguration returns a boolean if a field has been set.

### GetMetadata

`func (o *ComponentsNode) GetMetadata() map[string]interface{}`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *ComponentsNode) GetMetadataOk() (*map[string]interface{}, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *ComponentsNode) SetMetadata(v map[string]interface{})`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *ComponentsNode) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetPosition

`func (o *ComponentsNode) GetPosition() ComponentsPosition`

GetPosition returns the Position field if non-nil, zero value otherwise.

### GetPositionOk

`func (o *ComponentsNode) GetPositionOk() (*ComponentsPosition, bool)`

GetPositionOk returns a tuple with the Position field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPosition

`func (o *ComponentsNode) SetPosition(v ComponentsPosition)`

SetPosition sets Position field to given value.

### HasPosition

`func (o *ComponentsNode) HasPosition() bool`

HasPosition returns a boolean if a field has been set.

### GetComponent

`func (o *ComponentsNode) GetComponent() NodeComponentRef`

GetComponent returns the Component field if non-nil, zero value otherwise.

### GetComponentOk

`func (o *ComponentsNode) GetComponentOk() (*NodeComponentRef, bool)`

GetComponentOk returns a tuple with the Component field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetComponent

`func (o *ComponentsNode) SetComponent(v NodeComponentRef)`

SetComponent sets Component field to given value.

### HasComponent

`func (o *ComponentsNode) HasComponent() bool`

HasComponent returns a boolean if a field has been set.

### GetBlueprint

`func (o *ComponentsNode) GetBlueprint() NodeBlueprintRef`

GetBlueprint returns the Blueprint field if non-nil, zero value otherwise.

### GetBlueprintOk

`func (o *ComponentsNode) GetBlueprintOk() (*NodeBlueprintRef, bool)`

GetBlueprintOk returns a tuple with the Blueprint field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBlueprint

`func (o *ComponentsNode) SetBlueprint(v NodeBlueprintRef)`

SetBlueprint sets Blueprint field to given value.

### HasBlueprint

`func (o *ComponentsNode) HasBlueprint() bool`

HasBlueprint returns a boolean if a field has been set.

### GetTrigger

`func (o *ComponentsNode) GetTrigger() NodeTriggerRef`

GetTrigger returns the Trigger field if non-nil, zero value otherwise.

### GetTriggerOk

`func (o *ComponentsNode) GetTriggerOk() (*NodeTriggerRef, bool)`

GetTriggerOk returns a tuple with the Trigger field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTrigger

`func (o *ComponentsNode) SetTrigger(v NodeTriggerRef)`

SetTrigger sets Trigger field to given value.

### HasTrigger

`func (o *ComponentsNode) HasTrigger() bool`

HasTrigger returns a boolean if a field has been set.

### GetWidget

`func (o *ComponentsNode) GetWidget() NodeWidgetRef`

GetWidget returns the Widget field if non-nil, zero value otherwise.

### GetWidgetOk

`func (o *ComponentsNode) GetWidgetOk() (*NodeWidgetRef, bool)`

GetWidgetOk returns a tuple with the Widget field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWidget

`func (o *ComponentsNode) SetWidget(v NodeWidgetRef)`

SetWidget sets Widget field to given value.

### HasWidget

`func (o *ComponentsNode) HasWidget() bool`

HasWidget returns a boolean if a field has been set.

### GetIsCollapsed

`func (o *ComponentsNode) GetIsCollapsed() bool`

GetIsCollapsed returns the IsCollapsed field if non-nil, zero value otherwise.

### GetIsCollapsedOk

`func (o *ComponentsNode) GetIsCollapsedOk() (*bool, bool)`

GetIsCollapsedOk returns a tuple with the IsCollapsed field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIsCollapsed

`func (o *ComponentsNode) SetIsCollapsed(v bool)`

SetIsCollapsed sets IsCollapsed field to given value.

### HasIsCollapsed

`func (o *ComponentsNode) HasIsCollapsed() bool`

HasIsCollapsed returns a boolean if a field has been set.

### GetIntegration

`func (o *ComponentsNode) GetIntegration() ComponentsIntegrationRef`

GetIntegration returns the Integration field if non-nil, zero value otherwise.

### GetIntegrationOk

`func (o *ComponentsNode) GetIntegrationOk() (*ComponentsIntegrationRef, bool)`

GetIntegrationOk returns a tuple with the Integration field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIntegration

`func (o *ComponentsNode) SetIntegration(v ComponentsIntegrationRef)`

SetIntegration sets Integration field to given value.

### HasIntegration

`func (o *ComponentsNode) HasIntegration() bool`

HasIntegration returns a boolean if a field has been set.

### GetErrorMessage

`func (o *ComponentsNode) GetErrorMessage() string`

GetErrorMessage returns the ErrorMessage field if non-nil, zero value otherwise.

### GetErrorMessageOk

`func (o *ComponentsNode) GetErrorMessageOk() (*string, bool)`

GetErrorMessageOk returns a tuple with the ErrorMessage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetErrorMessage

`func (o *ComponentsNode) SetErrorMessage(v string)`

SetErrorMessage sets ErrorMessage field to given value.

### HasErrorMessage

`func (o *ComponentsNode) HasErrorMessage() bool`

HasErrorMessage returns a boolean if a field has been set.

### GetWarningMessage

`func (o *ComponentsNode) GetWarningMessage() string`

GetWarningMessage returns the WarningMessage field if non-nil, zero value otherwise.

### GetWarningMessageOk

`func (o *ComponentsNode) GetWarningMessageOk() (*string, bool)`

GetWarningMessageOk returns a tuple with the WarningMessage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWarningMessage

`func (o *ComponentsNode) SetWarningMessage(v string)`

SetWarningMessage sets WarningMessage field to given value.

### HasWarningMessage

`func (o *ComponentsNode) HasWarningMessage() bool`

HasWarningMessage returns a boolean if a field has been set.

### GetPaused

`func (o *ComponentsNode) GetPaused() bool`

GetPaused returns the Paused field if non-nil, zero value otherwise.

### GetPausedOk

`func (o *ComponentsNode) GetPausedOk() (*bool, bool)`

GetPausedOk returns a tuple with the Paused field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPaused

`func (o *ComponentsNode) SetPaused(v bool)`

SetPaused sets Paused field to given value.

### HasPaused

`func (o *ComponentsNode) HasPaused() bool`

HasPaused returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


