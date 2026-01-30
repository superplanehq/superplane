# WorkflowsListChildExecutionsResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Executions** | Pointer to [**[]WorkflowsWorkflowNodeExecution**](WorkflowsWorkflowNodeExecution.md) |  | [optional] 

## Methods

### NewWorkflowsListChildExecutionsResponse

`func NewWorkflowsListChildExecutionsResponse() *WorkflowsListChildExecutionsResponse`

NewWorkflowsListChildExecutionsResponse instantiates a new WorkflowsListChildExecutionsResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWorkflowsListChildExecutionsResponseWithDefaults

`func NewWorkflowsListChildExecutionsResponseWithDefaults() *WorkflowsListChildExecutionsResponse`

NewWorkflowsListChildExecutionsResponseWithDefaults instantiates a new WorkflowsListChildExecutionsResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetExecutions

`func (o *WorkflowsListChildExecutionsResponse) GetExecutions() []WorkflowsWorkflowNodeExecution`

GetExecutions returns the Executions field if non-nil, zero value otherwise.

### GetExecutionsOk

`func (o *WorkflowsListChildExecutionsResponse) GetExecutionsOk() (*[]WorkflowsWorkflowNodeExecution, bool)`

GetExecutionsOk returns a tuple with the Executions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExecutions

`func (o *WorkflowsListChildExecutionsResponse) SetExecutions(v []WorkflowsWorkflowNodeExecution)`

SetExecutions sets Executions field to given value.

### HasExecutions

`func (o *WorkflowsListChildExecutionsResponse) HasExecutions() bool`

HasExecutions returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


