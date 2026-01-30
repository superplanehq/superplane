# WorkflowsWorkflowNodeQueueItem

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**WorkflowId** | Pointer to **string** |  | [optional] 
**NodeId** | Pointer to **string** |  | [optional] 
**Input** | Pointer to **map[string]interface{}** |  | [optional] 
**RootEvent** | Pointer to [**WorkflowsWorkflowEvent**](WorkflowsWorkflowEvent.md) |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 

## Methods

### NewWorkflowsWorkflowNodeQueueItem

`func NewWorkflowsWorkflowNodeQueueItem() *WorkflowsWorkflowNodeQueueItem`

NewWorkflowsWorkflowNodeQueueItem instantiates a new WorkflowsWorkflowNodeQueueItem object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWorkflowsWorkflowNodeQueueItemWithDefaults

`func NewWorkflowsWorkflowNodeQueueItemWithDefaults() *WorkflowsWorkflowNodeQueueItem`

NewWorkflowsWorkflowNodeQueueItemWithDefaults instantiates a new WorkflowsWorkflowNodeQueueItem object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *WorkflowsWorkflowNodeQueueItem) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *WorkflowsWorkflowNodeQueueItem) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *WorkflowsWorkflowNodeQueueItem) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *WorkflowsWorkflowNodeQueueItem) HasId() bool`

HasId returns a boolean if a field has been set.

### GetWorkflowId

`func (o *WorkflowsWorkflowNodeQueueItem) GetWorkflowId() string`

GetWorkflowId returns the WorkflowId field if non-nil, zero value otherwise.

### GetWorkflowIdOk

`func (o *WorkflowsWorkflowNodeQueueItem) GetWorkflowIdOk() (*string, bool)`

GetWorkflowIdOk returns a tuple with the WorkflowId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWorkflowId

`func (o *WorkflowsWorkflowNodeQueueItem) SetWorkflowId(v string)`

SetWorkflowId sets WorkflowId field to given value.

### HasWorkflowId

`func (o *WorkflowsWorkflowNodeQueueItem) HasWorkflowId() bool`

HasWorkflowId returns a boolean if a field has been set.

### GetNodeId

`func (o *WorkflowsWorkflowNodeQueueItem) GetNodeId() string`

GetNodeId returns the NodeId field if non-nil, zero value otherwise.

### GetNodeIdOk

`func (o *WorkflowsWorkflowNodeQueueItem) GetNodeIdOk() (*string, bool)`

GetNodeIdOk returns a tuple with the NodeId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNodeId

`func (o *WorkflowsWorkflowNodeQueueItem) SetNodeId(v string)`

SetNodeId sets NodeId field to given value.

### HasNodeId

`func (o *WorkflowsWorkflowNodeQueueItem) HasNodeId() bool`

HasNodeId returns a boolean if a field has been set.

### GetInput

`func (o *WorkflowsWorkflowNodeQueueItem) GetInput() map[string]interface{}`

GetInput returns the Input field if non-nil, zero value otherwise.

### GetInputOk

`func (o *WorkflowsWorkflowNodeQueueItem) GetInputOk() (*map[string]interface{}, bool)`

GetInputOk returns a tuple with the Input field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInput

`func (o *WorkflowsWorkflowNodeQueueItem) SetInput(v map[string]interface{})`

SetInput sets Input field to given value.

### HasInput

`func (o *WorkflowsWorkflowNodeQueueItem) HasInput() bool`

HasInput returns a boolean if a field has been set.

### GetRootEvent

`func (o *WorkflowsWorkflowNodeQueueItem) GetRootEvent() WorkflowsWorkflowEvent`

GetRootEvent returns the RootEvent field if non-nil, zero value otherwise.

### GetRootEventOk

`func (o *WorkflowsWorkflowNodeQueueItem) GetRootEventOk() (*WorkflowsWorkflowEvent, bool)`

GetRootEventOk returns a tuple with the RootEvent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRootEvent

`func (o *WorkflowsWorkflowNodeQueueItem) SetRootEvent(v WorkflowsWorkflowEvent)`

SetRootEvent sets RootEvent field to given value.

### HasRootEvent

`func (o *WorkflowsWorkflowNodeQueueItem) HasRootEvent() bool`

HasRootEvent returns a boolean if a field has been set.

### GetCreatedAt

`func (o *WorkflowsWorkflowNodeQueueItem) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *WorkflowsWorkflowNodeQueueItem) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *WorkflowsWorkflowNodeQueueItem) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *WorkflowsWorkflowNodeQueueItem) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


