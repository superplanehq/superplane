# WorkflowsWorkflowEventWithExecutions

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** |  | [optional] 
**WorkflowId** | Pointer to **string** |  | [optional] 
**NodeId** | Pointer to **string** |  | [optional] 
**Channel** | Pointer to **string** |  | [optional] 
**Data** | Pointer to **map[string]interface{}** |  | [optional] 
**CreatedAt** | Pointer to **time.Time** |  | [optional] 
**Executions** | Pointer to [**[]WorkflowsWorkflowNodeExecution**](WorkflowsWorkflowNodeExecution.md) |  | [optional] 
**CustomName** | Pointer to **string** |  | [optional] 

## Methods

### NewWorkflowsWorkflowEventWithExecutions

`func NewWorkflowsWorkflowEventWithExecutions() *WorkflowsWorkflowEventWithExecutions`

NewWorkflowsWorkflowEventWithExecutions instantiates a new WorkflowsWorkflowEventWithExecutions object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWorkflowsWorkflowEventWithExecutionsWithDefaults

`func NewWorkflowsWorkflowEventWithExecutionsWithDefaults() *WorkflowsWorkflowEventWithExecutions`

NewWorkflowsWorkflowEventWithExecutionsWithDefaults instantiates a new WorkflowsWorkflowEventWithExecutions object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *WorkflowsWorkflowEventWithExecutions) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *WorkflowsWorkflowEventWithExecutions) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *WorkflowsWorkflowEventWithExecutions) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *WorkflowsWorkflowEventWithExecutions) HasId() bool`

HasId returns a boolean if a field has been set.

### GetWorkflowId

`func (o *WorkflowsWorkflowEventWithExecutions) GetWorkflowId() string`

GetWorkflowId returns the WorkflowId field if non-nil, zero value otherwise.

### GetWorkflowIdOk

`func (o *WorkflowsWorkflowEventWithExecutions) GetWorkflowIdOk() (*string, bool)`

GetWorkflowIdOk returns a tuple with the WorkflowId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWorkflowId

`func (o *WorkflowsWorkflowEventWithExecutions) SetWorkflowId(v string)`

SetWorkflowId sets WorkflowId field to given value.

### HasWorkflowId

`func (o *WorkflowsWorkflowEventWithExecutions) HasWorkflowId() bool`

HasWorkflowId returns a boolean if a field has been set.

### GetNodeId

`func (o *WorkflowsWorkflowEventWithExecutions) GetNodeId() string`

GetNodeId returns the NodeId field if non-nil, zero value otherwise.

### GetNodeIdOk

`func (o *WorkflowsWorkflowEventWithExecutions) GetNodeIdOk() (*string, bool)`

GetNodeIdOk returns a tuple with the NodeId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNodeId

`func (o *WorkflowsWorkflowEventWithExecutions) SetNodeId(v string)`

SetNodeId sets NodeId field to given value.

### HasNodeId

`func (o *WorkflowsWorkflowEventWithExecutions) HasNodeId() bool`

HasNodeId returns a boolean if a field has been set.

### GetChannel

`func (o *WorkflowsWorkflowEventWithExecutions) GetChannel() string`

GetChannel returns the Channel field if non-nil, zero value otherwise.

### GetChannelOk

`func (o *WorkflowsWorkflowEventWithExecutions) GetChannelOk() (*string, bool)`

GetChannelOk returns a tuple with the Channel field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetChannel

`func (o *WorkflowsWorkflowEventWithExecutions) SetChannel(v string)`

SetChannel sets Channel field to given value.

### HasChannel

`func (o *WorkflowsWorkflowEventWithExecutions) HasChannel() bool`

HasChannel returns a boolean if a field has been set.

### GetData

`func (o *WorkflowsWorkflowEventWithExecutions) GetData() map[string]interface{}`

GetData returns the Data field if non-nil, zero value otherwise.

### GetDataOk

`func (o *WorkflowsWorkflowEventWithExecutions) GetDataOk() (*map[string]interface{}, bool)`

GetDataOk returns a tuple with the Data field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetData

`func (o *WorkflowsWorkflowEventWithExecutions) SetData(v map[string]interface{})`

SetData sets Data field to given value.

### HasData

`func (o *WorkflowsWorkflowEventWithExecutions) HasData() bool`

HasData returns a boolean if a field has been set.

### GetCreatedAt

`func (o *WorkflowsWorkflowEventWithExecutions) GetCreatedAt() time.Time`

GetCreatedAt returns the CreatedAt field if non-nil, zero value otherwise.

### GetCreatedAtOk

`func (o *WorkflowsWorkflowEventWithExecutions) GetCreatedAtOk() (*time.Time, bool)`

GetCreatedAtOk returns a tuple with the CreatedAt field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCreatedAt

`func (o *WorkflowsWorkflowEventWithExecutions) SetCreatedAt(v time.Time)`

SetCreatedAt sets CreatedAt field to given value.

### HasCreatedAt

`func (o *WorkflowsWorkflowEventWithExecutions) HasCreatedAt() bool`

HasCreatedAt returns a boolean if a field has been set.

### GetExecutions

`func (o *WorkflowsWorkflowEventWithExecutions) GetExecutions() []WorkflowsWorkflowNodeExecution`

GetExecutions returns the Executions field if non-nil, zero value otherwise.

### GetExecutionsOk

`func (o *WorkflowsWorkflowEventWithExecutions) GetExecutionsOk() (*[]WorkflowsWorkflowNodeExecution, bool)`

GetExecutionsOk returns a tuple with the Executions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExecutions

`func (o *WorkflowsWorkflowEventWithExecutions) SetExecutions(v []WorkflowsWorkflowNodeExecution)`

SetExecutions sets Executions field to given value.

### HasExecutions

`func (o *WorkflowsWorkflowEventWithExecutions) HasExecutions() bool`

HasExecutions returns a boolean if a field has been set.

### GetCustomName

`func (o *WorkflowsWorkflowEventWithExecutions) GetCustomName() string`

GetCustomName returns the CustomName field if non-nil, zero value otherwise.

### GetCustomNameOk

`func (o *WorkflowsWorkflowEventWithExecutions) GetCustomNameOk() (*string, bool)`

GetCustomNameOk returns a tuple with the CustomName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCustomName

`func (o *WorkflowsWorkflowEventWithExecutions) SetCustomName(v string)`

SetCustomName sets CustomName field to given value.

### HasCustomName

`func (o *WorkflowsWorkflowEventWithExecutions) HasCustomName() bool`

HasCustomName returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


