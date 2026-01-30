# WorkflowsWorkflowStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**LastExecutions** | Pointer to [**[]WorkflowsWorkflowNodeExecution**](WorkflowsWorkflowNodeExecution.md) |  | [optional] 
**NextQueueItems** | Pointer to [**[]WorkflowsWorkflowNodeQueueItem**](WorkflowsWorkflowNodeQueueItem.md) |  | [optional] 
**LastEvents** | Pointer to [**[]WorkflowsWorkflowEvent**](WorkflowsWorkflowEvent.md) |  | [optional] 

## Methods

### NewWorkflowsWorkflowStatus

`func NewWorkflowsWorkflowStatus() *WorkflowsWorkflowStatus`

NewWorkflowsWorkflowStatus instantiates a new WorkflowsWorkflowStatus object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWorkflowsWorkflowStatusWithDefaults

`func NewWorkflowsWorkflowStatusWithDefaults() *WorkflowsWorkflowStatus`

NewWorkflowsWorkflowStatusWithDefaults instantiates a new WorkflowsWorkflowStatus object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetLastExecutions

`func (o *WorkflowsWorkflowStatus) GetLastExecutions() []WorkflowsWorkflowNodeExecution`

GetLastExecutions returns the LastExecutions field if non-nil, zero value otherwise.

### GetLastExecutionsOk

`func (o *WorkflowsWorkflowStatus) GetLastExecutionsOk() (*[]WorkflowsWorkflowNodeExecution, bool)`

GetLastExecutionsOk returns a tuple with the LastExecutions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastExecutions

`func (o *WorkflowsWorkflowStatus) SetLastExecutions(v []WorkflowsWorkflowNodeExecution)`

SetLastExecutions sets LastExecutions field to given value.

### HasLastExecutions

`func (o *WorkflowsWorkflowStatus) HasLastExecutions() bool`

HasLastExecutions returns a boolean if a field has been set.

### GetNextQueueItems

`func (o *WorkflowsWorkflowStatus) GetNextQueueItems() []WorkflowsWorkflowNodeQueueItem`

GetNextQueueItems returns the NextQueueItems field if non-nil, zero value otherwise.

### GetNextQueueItemsOk

`func (o *WorkflowsWorkflowStatus) GetNextQueueItemsOk() (*[]WorkflowsWorkflowNodeQueueItem, bool)`

GetNextQueueItemsOk returns a tuple with the NextQueueItems field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNextQueueItems

`func (o *WorkflowsWorkflowStatus) SetNextQueueItems(v []WorkflowsWorkflowNodeQueueItem)`

SetNextQueueItems sets NextQueueItems field to given value.

### HasNextQueueItems

`func (o *WorkflowsWorkflowStatus) HasNextQueueItems() bool`

HasNextQueueItems returns a boolean if a field has been set.

### GetLastEvents

`func (o *WorkflowsWorkflowStatus) GetLastEvents() []WorkflowsWorkflowEvent`

GetLastEvents returns the LastEvents field if non-nil, zero value otherwise.

### GetLastEventsOk

`func (o *WorkflowsWorkflowStatus) GetLastEventsOk() (*[]WorkflowsWorkflowEvent, bool)`

GetLastEventsOk returns a tuple with the LastEvents field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastEvents

`func (o *WorkflowsWorkflowStatus) SetLastEvents(v []WorkflowsWorkflowEvent)`

SetLastEvents sets LastEvents field to given value.

### HasLastEvents

`func (o *WorkflowsWorkflowStatus) HasLastEvents() bool`

HasLastEvents returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


