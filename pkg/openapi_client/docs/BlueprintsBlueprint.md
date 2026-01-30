# BlueprintsBlueprint

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**OrganizationId** | Pointer to **string** |  | [optional] 
**Name** | Pointer to **string** |  | [optional] 
**Description** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**Nodes** | Pointer to [**[]ComponentsNode**](ComponentsNode.md) |  | [optional] 
**Edges** | Pointer to [**[]ComponentsEdge**](ComponentsEdge.md) |  | [optional] 
**Configuration** | Pointer to [**[]ConfigurationField**](ConfigurationField.md) |  | [optional] 
**OutputChannels** | Pointer to [**[]SuperplaneBlueprintsOutputChannel**](SuperplaneBlueprintsOutputChannel.md) |  | [optional] 
**Icon** | Pointer to **string** |  | [optional] 
**Color** | Pointer to **string** |  | [optional] 
**CreatedBy** | Pointer to [**SuperplaneBlueprintsUserRef**](SuperplaneBlueprintsUserRef.md) |  | [optional] 

## Methods

### NewBlueprintsBlueprint

`func NewBlueprintsBlueprint() *BlueprintsBlueprint`

NewBlueprintsBlueprint instantiates a new BlueprintsBlueprint object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewBlueprintsBlueprintWithDefaults

`func NewBlueprintsBlueprintWithDefaults() *BlueprintsBlueprint`

NewBlueprintsBlueprintWithDefaults instantiates a new BlueprintsBlueprint object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *BlueprintsBlueprint) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *BlueprintsBlueprint) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *BlueprintsBlueprint) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *BlueprintsBlueprint) HasId() bool`

HasId returns a boolean if a field has been set.

### GetOrganizationId

`func (o *BlueprintsBlueprint) GetOrganizationId() string`

GetOrganizationId returns the OrganizationId field if non-nil, zero value otherwise.

### GetOrganizationIdOk

`func (o *BlueprintsBlueprint) GetOrganizationIdOk() (*string, bool)`

GetOrganizationIdOk returns a tuple with the OrganizationId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganizationId

`func (o *BlueprintsBlueprint) SetOrganizationId(v string)`

SetOrganizationId sets OrganizationId field to given value.

### HasOrganizationId

`func (o *BlueprintsBlueprint) HasOrganizationId() bool`

HasOrganizationId returns a boolean if a field has been set.

### GetName

`func (o *BlueprintsBlueprint) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *BlueprintsBlueprint) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *BlueprintsBlueprint) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *BlueprintsBlueprint) HasName() bool`

HasName returns a boolean if a field has been set.

### GetDescription

`func (o *BlueprintsBlueprint) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *BlueprintsBlueprint) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *BlueprintsBlueprint) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *BlueprintsBlueprint) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetCreatedAt

`func (o *BlueprintsBlueprint) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *BlueprintsBlueprint) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *BlueprintsBlueprint) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *BlueprintsBlueprint) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *BlueprintsBlueprint) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *BlueprintsBlueprint) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *BlueprintsBlueprint) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *BlueprintsBlueprint) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetNodes

`func (o *BlueprintsBlueprint) GetNodes() []ComponentsNode`

GetNodes returns the Nodes field if non-nil, zero value otherwise.

### GetNodesOk

`func (o *BlueprintsBlueprint) GetNodesOk() (*[]ComponentsNode, bool)`

GetNodesOk returns a tuple with the Nodes field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNodes

`func (o *BlueprintsBlueprint) SetNodes(v []ComponentsNode)`

SetNodes sets Nodes field to given value.

### HasNodes

`func (o *BlueprintsBlueprint) HasNodes() bool`

HasNodes returns a boolean if a field has been set.

### GetEdges

`func (o *BlueprintsBlueprint) GetEdges() []ComponentsEdge`

GetEdges returns the Edges field if non-nil, zero value otherwise.

### GetEdgesOk

`func (o *BlueprintsBlueprint) GetEdgesOk() (*[]ComponentsEdge, bool)`

GetEdgesOk returns a tuple with the Edges field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEdges

`func (o *BlueprintsBlueprint) SetEdges(v []ComponentsEdge)`

SetEdges sets Edges field to given value.

### HasEdges

`func (o *BlueprintsBlueprint) HasEdges() bool`

HasEdges returns a boolean if a field has been set.

### GetConfiguration

`func (o *BlueprintsBlueprint) GetConfiguration() []ConfigurationField`

GetConfiguration returns the Configuration field if non-nil, zero value otherwise.

### GetConfigurationOk

`func (o *BlueprintsBlueprint) GetConfigurationOk() (*[]ConfigurationField, bool)`

GetConfigurationOk returns a tuple with the Configuration field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfiguration

`func (o *BlueprintsBlueprint) SetConfiguration(v []ConfigurationField)`

SetConfiguration sets Configuration field to given value.

### HasConfiguration

`func (o *BlueprintsBlueprint) HasConfiguration() bool`

HasConfiguration returns a boolean if a field has been set.

### GetOutputChannels

`func (o *BlueprintsBlueprint) GetOutputChannels() []SuperplaneBlueprintsOutputChannel`

GetOutputChannels returns the OutputChannels field if non-nil, zero value otherwise.

### GetOutputChannelsOk

`func (o *BlueprintsBlueprint) GetOutputChannelsOk() (*[]SuperplaneBlueprintsOutputChannel, bool)`

GetOutputChannelsOk returns a tuple with the OutputChannels field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOutputChannels

`func (o *BlueprintsBlueprint) SetOutputChannels(v []SuperplaneBlueprintsOutputChannel)`

SetOutputChannels sets OutputChannels field to given value.

### HasOutputChannels

`func (o *BlueprintsBlueprint) HasOutputChannels() bool`

HasOutputChannels returns a boolean if a field has been set.

### GetIcon

`func (o *BlueprintsBlueprint) GetIcon() string`

GetIcon returns the Icon field if non-nil, zero value otherwise.

### GetIconOk

`func (o *BlueprintsBlueprint) GetIconOk() (*string, bool)`

GetIconOk returns a tuple with the Icon field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIcon

`func (o *BlueprintsBlueprint) SetIcon(v string)`

SetIcon sets Icon field to given value.

### HasIcon

`func (o *BlueprintsBlueprint) HasIcon() bool`

HasIcon returns a boolean if a field has been set.

### GetColor

`func (o *BlueprintsBlueprint) GetColor() string`

GetColor returns the Color field if non-nil, zero value otherwise.

### GetColorOk

`func (o *BlueprintsBlueprint) GetColorOk() (*string, bool)`

GetColorOk returns a tuple with the Color field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetColor

`func (o *BlueprintsBlueprint) SetColor(v string)`

SetColor sets Color field to given value.

### HasColor

`func (o *BlueprintsBlueprint) HasColor() bool`

HasColor returns a boolean if a field has been set.

### GetCreatedBy

`func (o *BlueprintsBlueprint) GetCreatedBy() SuperplaneBlueprintsUserRef`

GetCreatedBy returns the CreatedBy field if non-nil, zero value otherwise.

### GetCreatedByOk

`func (o *BlueprintsBlueprint) GetCreatedByOk() (*SuperplaneBlueprintsUserRef, bool)`

GetCreatedByOk returns a tuple with the CreatedBy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedBy

`func (o *BlueprintsBlueprint) SetCreatedBy(v SuperplaneBlueprintsUserRef)`

SetCreatedBy sets CreatedBy field to given value.

### HasCreatedBy

`func (o *BlueprintsBlueprint) HasCreatedBy() bool`

HasCreatedBy returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


