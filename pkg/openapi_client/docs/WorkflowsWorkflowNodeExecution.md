# WorkflowsWorkflowNodeExecution

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**WorkflowId** | Pointer to **string** |  | [optional] 
**NodeId** | Pointer to **string** |  | [optional] 
**ParentExecutionId** | Pointer to **string** |  | [optional] 
**PreviousExecutionId** | Pointer to **string** |  | [optional] 
**State** | Pointer to [**WorkflowNodeExecutionState**](WorkflowNodeExecutionState.md) |  | [optional] [default to WORKFLOWNODEEXECUTIONSTATE_STATE_UNKNOWN]
**Result** | Pointer to [**WorkflowNodeExecutionResult**](WorkflowNodeExecutionResult.md) |  | [optional] [default to WORKFLOWNODEEXECUTIONRESULT_RESULT_UNKNOWN]
**ResultReason** | Pointer to [**WorkflowNodeExecutionResultReason**](WorkflowNodeExecutionResultReason.md) |  | [optional] [default to WORKFLOWNODEEXECUTIONRESULTREASON_RESULT_REASON_OK]
**ResultMessage** | Pointer to **string** |  | [optional] 
**Input** | Pointer to **map[string]interface{}** |  | [optional] 
**Outputs** | Pointer to **map[string]interface{}** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**UpdatedAt** | Pointer to **time.Time** |  | [optional] 
**Metadata** | Pointer to **map[string]interface{}** |  | [optional] 
**Configuration** | Pointer to **map[string]interface{}** |  | [optional] 
**ChildExecutions** | Pointer to [**[]WorkflowsWorkflowNodeExecution**](WorkflowsWorkflowNodeExecution.md) |  | [optional] 
**RootEvent** | Pointer to [**WorkflowsWorkflowEvent**](WorkflowsWorkflowEvent.md) |  | [optional] 
**CancelledBy** | Pointer to [**SuperplaneWorkflowsUserRef**](SuperplaneWorkflowsUserRef.md) |  | [optional] 

## Methods

### NewWorkflowsWorkflowNodeExecution

`func NewWorkflowsWorkflowNodeExecution() *WorkflowsWorkflowNodeExecution`

NewWorkflowsWorkflowNodeExecution instantiates a new WorkflowsWorkflowNodeExecution object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWorkflowsWorkflowNodeExecutionWithDefaults

`func NewWorkflowsWorkflowNodeExecutionWithDefaults() *WorkflowsWorkflowNodeExecution`

NewWorkflowsWorkflowNodeExecutionWithDefaults instantiates a new WorkflowsWorkflowNodeExecution object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *WorkflowsWorkflowNodeExecution) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *WorkflowsWorkflowNodeExecution) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *WorkflowsWorkflowNodeExecution) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *WorkflowsWorkflowNodeExecution) HasId() bool`

HasId returns a boolean if a field has been set.

### GetWorkflowId

`func (o *WorkflowsWorkflowNodeExecution) GetWorkflowId() string`

GetWorkflowId returns the WorkflowId field if non-nil, zero value otherwise.

### GetWorkflowIdOk

`func (o *WorkflowsWorkflowNodeExecution) GetWorkflowIdOk() (*string, bool)`

GetWorkflowIdOk returns a tuple with the WorkflowId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWorkflowId

`func (o *WorkflowsWorkflowNodeExecution) SetWorkflowId(v string)`

SetWorkflowId sets WorkflowId field to given value.

### HasWorkflowId

`func (o *WorkflowsWorkflowNodeExecution) HasWorkflowId() bool`

HasWorkflowId returns a boolean if a field has been set.

### GetNodeId

`func (o *WorkflowsWorkflowNodeExecution) GetNodeId() string`

GetNodeId returns the NodeId field if non-nil, zero value otherwise.

### GetNodeIdOk

`func (o *WorkflowsWorkflowNodeExecution) GetNodeIdOk() (*string, bool)`

GetNodeIdOk returns a tuple with the NodeId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNodeId

`func (o *WorkflowsWorkflowNodeExecution) SetNodeId(v string)`

SetNodeId sets NodeId field to given value.

### HasNodeId

`func (o *WorkflowsWorkflowNodeExecution) HasNodeId() bool`

HasNodeId returns a boolean if a field has been set.

### GetParentExecutionId

`func (o *WorkflowsWorkflowNodeExecution) GetParentExecutionId() string`

GetParentExecutionId returns the ParentExecutionId field if non-nil, zero value otherwise.

### GetParentExecutionIdOk

`func (o *WorkflowsWorkflowNodeExecution) GetParentExecutionIdOk() (*string, bool)`

GetParentExecutionIdOk returns a tuple with the ParentExecutionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetParentExecutionId

`func (o *WorkflowsWorkflowNodeExecution) SetParentExecutionId(v string)`

SetParentExecutionId sets ParentExecutionId field to given value.

### HasParentExecutionId

`func (o *WorkflowsWorkflowNodeExecution) HasParentExecutionId() bool`

HasParentExecutionId returns a boolean if a field has been set.

### GetPreviousExecutionId

`func (o *WorkflowsWorkflowNodeExecution) GetPreviousExecutionId() string`

GetPreviousExecutionId returns the PreviousExecutionId field if non-nil, zero value otherwise.

### GetPreviousExecutionIdOk

`func (o *WorkflowsWorkflowNodeExecution) GetPreviousExecutionIdOk() (*string, bool)`

GetPreviousExecutionIdOk returns a tuple with the PreviousExecutionId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPreviousExecutionId

`func (o *WorkflowsWorkflowNodeExecution) SetPreviousExecutionId(v string)`

SetPreviousExecutionId sets PreviousExecutionId field to given value.

### HasPreviousExecutionId

`func (o *WorkflowsWorkflowNodeExecution) HasPreviousExecutionId() bool`

HasPreviousExecutionId returns a boolean if a field has been set.

### GetState

`func (o *WorkflowsWorkflowNodeExecution) GetState() WorkflowNodeExecutionState`

GetState returns the State field if non-nil, zero value otherwise.

### GetStateOk

`func (o *WorkflowsWorkflowNodeExecution) GetStateOk() (*WorkflowNodeExecutionState, bool)`

GetStateOk returns a tuple with the State field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetState

`func (o *WorkflowsWorkflowNodeExecution) SetState(v WorkflowNodeExecutionState)`

SetState sets State field to given value.

### HasState

`func (o *WorkflowsWorkflowNodeExecution) HasState() bool`

HasState returns a boolean if a field has been set.

### GetResult

`func (o *WorkflowsWorkflowNodeExecution) GetResult() WorkflowNodeExecutionResult`

GetResult returns the Result field if non-nil, zero value otherwise.

### GetResultOk

`func (o *WorkflowsWorkflowNodeExecution) GetResultOk() (*WorkflowNodeExecutionResult, bool)`

GetResultOk returns a tuple with the Result field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResult

`func (o *WorkflowsWorkflowNodeExecution) SetResult(v WorkflowNodeExecutionResult)`

SetResult sets Result field to given value.

### HasResult

`func (o *WorkflowsWorkflowNodeExecution) HasResult() bool`

HasResult returns a boolean if a field has been set.

### GetResultReason

`func (o *WorkflowsWorkflowNodeExecution) GetResultReason() WorkflowNodeExecutionResultReason`

GetResultReason returns the ResultReason field if non-nil, zero value otherwise.

### GetResultReasonOk

`func (o *WorkflowsWorkflowNodeExecution) GetResultReasonOk() (*WorkflowNodeExecutionResultReason, bool)`

GetResultReasonOk returns a tuple with the ResultReason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResultReason

`func (o *WorkflowsWorkflowNodeExecution) SetResultReason(v WorkflowNodeExecutionResultReason)`

SetResultReason sets ResultReason field to given value.

### HasResultReason

`func (o *WorkflowsWorkflowNodeExecution) HasResultReason() bool`

HasResultReason returns a boolean if a field has been set.

### GetResultMessage

`func (o *WorkflowsWorkflowNodeExecution) GetResultMessage() string`

GetResultMessage returns the ResultMessage field if non-nil, zero value otherwise.

### GetResultMessageOk

`func (o *WorkflowsWorkflowNodeExecution) GetResultMessageOk() (*string, bool)`

GetResultMessageOk returns a tuple with the ResultMessage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResultMessage

`func (o *WorkflowsWorkflowNodeExecution) SetResultMessage(v string)`

SetResultMessage sets ResultMessage field to given value.

### HasResultMessage

`func (o *WorkflowsWorkflowNodeExecution) HasResultMessage() bool`

HasResultMessage returns a boolean if a field has been set.

### GetInput

`func (o *WorkflowsWorkflowNodeExecution) GetInput() map[string]interface{}`

GetInput returns the Input field if non-nil, zero value otherwise.

### GetInputOk

`func (o *WorkflowsWorkflowNodeExecution) GetInputOk() (*map[string]interface{}, bool)`

GetInputOk returns a tuple with the Input field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInput

`func (o *WorkflowsWorkflowNodeExecution) SetInput(v map[string]interface{})`

SetInput sets Input field to given value.

### HasInput

`func (o *WorkflowsWorkflowNodeExecution) HasInput() bool`

HasInput returns a boolean if a field has been set.

### GetOutputs

`func (o *WorkflowsWorkflowNodeExecution) GetOutputs() map[string]interface{}`

GetOutputs returns the Outputs field if non-nil, zero value otherwise.

### GetOutputsOk

`func (o *WorkflowsWorkflowNodeExecution) GetOutputsOk() (*map[string]interface{}, bool)`

GetOutputsOk returns a tuple with the Outputs field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOutputs

`func (o *WorkflowsWorkflowNodeExecution) SetOutputs(v map[string]interface{})`

SetOutputs sets Outputs field to given value.

### HasOutputs

`func (o *WorkflowsWorkflowNodeExecution) HasOutputs() bool`

HasOutputs returns a boolean if a field has been set.

### GetCreatedAt

`func (o *WorkflowsWorkflowNodeExecution) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *WorkflowsWorkflowNodeExecution) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *WorkflowsWorkflowNodeExecution) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *WorkflowsWorkflowNodeExecution) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetUpdatedAt

`func (o *WorkflowsWorkflowNodeExecution) GetUpdatedAt() time.Time`

GetUpdatedAt returns the UpdatedAt field if non-nil, zero value otherwise.

### GetUpdatedAtOk

`func (o *WorkflowsWorkflowNodeExecution) GetUpdatedAtOk() (*time.Time, bool)`

GetUpdatedAtOk returns a tuple with the UpdatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUpdatedAt

`func (o *WorkflowsWorkflowNodeExecution) SetUpdatedAt(v time.Time)`

SetUpdatedAt sets UpdatedAt field to given value.

### HasUpdatedAt

`func (o *WorkflowsWorkflowNodeExecution) HasUpdatedAt() bool`

HasUpdatedAt returns a boolean if a field has been set.

### GetMetadata

`func (o *WorkflowsWorkflowNodeExecution) GetMetadata() map[string]interface{}`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *WorkflowsWorkflowNodeExecution) GetMetadataOk() (*map[string]interface{}, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *WorkflowsWorkflowNodeExecution) SetMetadata(v map[string]interface{})`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *WorkflowsWorkflowNodeExecution) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetConfiguration

`func (o *WorkflowsWorkflowNodeExecution) GetConfiguration() map[string]interface{}`

GetConfiguration returns the Configuration field if non-nil, zero value otherwise.

### GetConfigurationOk

`func (o *WorkflowsWorkflowNodeExecution) GetConfigurationOk() (*map[string]interface{}, bool)`

GetConfigurationOk returns a tuple with the Configuration field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfiguration

`func (o *WorkflowsWorkflowNodeExecution) SetConfiguration(v map[string]interface{})`

SetConfiguration sets Configuration field to given value.

### HasConfiguration

`func (o *WorkflowsWorkflowNodeExecution) HasConfiguration() bool`

HasConfiguration returns a boolean if a field has been set.

### GetChildExecutions

`func (o *WorkflowsWorkflowNodeExecution) GetChildExecutions() []WorkflowsWorkflowNodeExecution`

GetChildExecutions returns the ChildExecutions field if non-nil, zero value otherwise.

### GetChildExecutionsOk

`func (o *WorkflowsWorkflowNodeExecution) GetChildExecutionsOk() (*[]WorkflowsWorkflowNodeExecution, bool)`

GetChildExecutionsOk returns a tuple with the ChildExecutions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChildExecutions

`func (o *WorkflowsWorkflowNodeExecution) SetChildExecutions(v []WorkflowsWorkflowNodeExecution)`

SetChildExecutions sets ChildExecutions field to given value.

### HasChildExecutions

`func (o *WorkflowsWorkflowNodeExecution) HasChildExecutions() bool`

HasChildExecutions returns a boolean if a field has been set.

### GetRootEvent

`func (o *WorkflowsWorkflowNodeExecution) GetRootEvent() WorkflowsWorkflowEvent`

GetRootEvent returns the RootEvent field if non-nil, zero value otherwise.

### GetRootEventOk

`func (o *WorkflowsWorkflowNodeExecution) GetRootEventOk() (*WorkflowsWorkflowEvent, bool)`

GetRootEventOk returns a tuple with the RootEvent field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRootEvent

`func (o *WorkflowsWorkflowNodeExecution) SetRootEvent(v WorkflowsWorkflowEvent)`

SetRootEvent sets RootEvent field to given value.

### HasRootEvent

`func (o *WorkflowsWorkflowNodeExecution) HasRootEvent() bool`

HasRootEvent returns a boolean if a field has been set.

### GetCancelledBy

`func (o *WorkflowsWorkflowNodeExecution) GetCancelledBy() SuperplaneWorkflowsUserRef`

GetCancelledBy returns the CancelledBy field if non-nil, zero value otherwise.

### GetCancelledByOk

`func (o *WorkflowsWorkflowNodeExecution) GetCancelledByOk() (*SuperplaneWorkflowsUserRef, bool)`

GetCancelledByOk returns a tuple with the CancelledBy field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCancelledBy

`func (o *WorkflowsWorkflowNodeExecution) SetCancelledBy(v SuperplaneWorkflowsUserRef)`

SetCancelledBy sets CancelledBy field to given value.

### HasCancelledBy

`func (o *WorkflowsWorkflowNodeExecution) HasCancelledBy() bool`

HasCancelledBy returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


