# WorkflowsWorkflowMetadata

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**OrganizationId** | Pointer to **string** |  | [optional] 
**Name** | Pointer to **string** |  | [optional] 
**Description** | Pointer to **string** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**CreatedBy** | Pointer to [**SuperplaneWorkflowsUserRef**](SuperplaneWorkflowsUserRef.md) |  | [optional] 
**IsTemplate** | Pointer to **bool** |  | [optional] 

## Methods

### NewWorkflowsWorkflowMetadata

`func NewWorkflowsWorkflowMetadata() *WorkflowsWorkflowMetadata`

NewWorkflowsWorkflowMetadata instantiates a new WorkflowsWorkflowMetadata object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWorkflowsWorkflowMetadataWithDefaults

`func NewWorkflowsWorkflowMetadataWithDefaults() *WorkflowsWorkflowMetadata`

NewWorkflowsWorkflowMetadataWithDefaults instantiates a new WorkflowsWorkflowMetadata object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *WorkflowsWorkflowMetadata) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *WorkflowsWorkflowMetadata) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *WorkflowsWorkflowMetadata) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *WorkflowsWorkflowMetadata) HasId() bool`

HasId returns a boolean if a field has been set.

### GetOrganizationId

`func (o *WorkflowsWorkflowMetadata) GetOrganizationId() string`

GetOrganizationId returns the OrganizationId field if non-nil, zero value otherwise.

### GetOrganizationIdOk

`func (o *WorkflowsWorkflowMetadata) GetOrganizationIdOk() (*string, bool)`

GetOrganizationIdOk returns a tuple with the OrganizationId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOrganizationId

`func (o *WorkflowsWorkflowMetadata) SetOrganizationId(v string)`

SetOrganizationId sets OrganizationId field to given value.

### HasOrganizationId

`func (o *WorkflowsWorkflowMetadata) HasOrganizationId() bool`

HasOrganizationId returns a boolean if a field has been set.

### GetName

`func (o *WorkflowsWorkflowMetadata) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *WorkflowsWorkflowMetadata) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *WorkflowsWorkflowMetadata) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *WorkflowsWorkflowMetadata) HasName() bool`

HasName returns a boolean if a field has been set.

### GetDescription

`func (o *WorkflowsWorkflowMetadata) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *WorkflowsWorkflowMetadata) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *WorkflowsWorkflowMetadata) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *WorkflowsWorkflowMetadata) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetCreatedAt

`func (o *WorkflowsWorkflowMetadata) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *WorkflowsWorkflowMetadata) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *WorkflowsWorkflowMetadata) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *WorkflowsWorkflowMetadata) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *WorkflowsWorkflowMetadata) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *WorkflowsWorkflowMetadata) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *WorkflowsWorkflowMetadata) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *WorkflowsWorkflowMetadata) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetCreatedBy

`func (o *WorkflowsWorkflowMetadata) GetCreatedBy() SuperplaneWorkflowsUserRef`

GetCreatedBy returns the CreatedBy field if non-nil, zero value otherwise.

### GetCreatedByOk

`func (o *WorkflowsWorkflowMetadata) GetCreatedByOk() (*SuperplaneWorkflowsUserRef, bool)`

GetCreatedByOk returns a tuple with the CreatedBy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedBy

`func (o *WorkflowsWorkflowMetadata) SetCreatedBy(v SuperplaneWorkflowsUserRef)`

SetCreatedBy sets CreatedBy field to given value.

### HasCreatedBy

`func (o *WorkflowsWorkflowMetadata) HasCreatedBy() bool`

HasCreatedBy returns a boolean if a field has been set.

### GetIsTemplate

`func (o *WorkflowsWorkflowMetadata) GetIsTemplate() bool`

GetIsTemplate returns the IsTemplate field if non-nil, zero value otherwise.

### GetIsTemplateOk

`func (o *WorkflowsWorkflowMetadata) GetIsTemplateOk() (*bool, bool)`

GetIsTemplateOk returns a tuple with the IsTemplate field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetIsTemplate

`func (o *WorkflowsWorkflowMetadata) SetIsTemplate(v bool)`

SetIsTemplate sets IsTemplate field to given value.

### HasIsTemplate

`func (o *WorkflowsWorkflowMetadata) HasIsTemplate() bool`

HasIsTemplate returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


