# WorkflowsListNodeExecutionsResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Executions** | Pointer to [**[]WorkflowsWorkflowNodeExecution**](WorkflowsWorkflowNodeExecution.md) |  | [optional] 
**TotalCount** | Pointer to **int64** |  | [optional] 
**HasNextPage** | Pointer to **bool** |  | [optional] 
**LastTimestamp** | Pointer to **time.Time** |  | [optional] 

## Methods

### NewWorkflowsListNodeExecutionsResponse

`func NewWorkflowsListNodeExecutionsResponse() *WorkflowsListNodeExecutionsResponse`

NewWorkflowsListNodeExecutionsResponse instantiates a new WorkflowsListNodeExecutionsResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWorkflowsListNodeExecutionsResponseWithDefaults

`func NewWorkflowsListNodeExecutionsResponseWithDefaults() *WorkflowsListNodeExecutionsResponse`

NewWorkflowsListNodeExecutionsResponseWithDefaults instantiates a new WorkflowsListNodeExecutionsResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetExecutions

`func (o *WorkflowsListNodeExecutionsResponse) GetExecutions() []WorkflowsWorkflowNodeExecution`

GetExecutions returns the Executions field if non-nil, zero value otherwise.

### GetExecutionsOk

`func (o *WorkflowsListNodeExecutionsResponse) GetExecutionsOk() (*[]WorkflowsWorkflowNodeExecution, bool)`

GetExecutionsOk returns a tuple with the Executions field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExecutions

`func (o *WorkflowsListNodeExecutionsResponse) SetExecutions(v []WorkflowsWorkflowNodeExecution)`

SetExecutions sets Executions field to given value.

### HasExecutions

`func (o *WorkflowsListNodeExecutionsResponse) HasExecutions() bool`

HasExecutions returns a boolean if a field has been set.

### GetTotalCount

`func (o *WorkflowsListNodeExecutionsResponse) GetTotalCount() int64`

GetTotalCount returns the TotalCount field if non-nil, zero value otherwise.

### GetTotalCountOk

`func (o *WorkflowsListNodeExecutionsResponse) GetTotalCountOk() (*int64, bool)`

GetTotalCountOk returns a tuple with the TotalCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotalCount

`func (o *WorkflowsListNodeExecutionsResponse) SetTotalCount(v int64)`

SetTotalCount sets TotalCount field to given value.

### HasTotalCount

`func (o *WorkflowsListNodeExecutionsResponse) HasTotalCount() bool`

HasTotalCount returns a boolean if a field has been set.

### GetHasNextPage

`func (o *WorkflowsListNodeExecutionsResponse) GetHasNextPage() bool`

GetHasNextPage returns the HasNextPage field if non-nil, zero value otherwise.

### GetHasNextPageOk

`func (o *WorkflowsListNodeExecutionsResponse) GetHasNextPageOk() (*bool, bool)`

GetHasNextPageOk returns a tuple with the HasNextPage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHasNextPage

`func (o *WorkflowsListNodeExecutionsResponse) SetHasNextPage(v bool)`

SetHasNextPage sets HasNextPage field to given value.

### HasHasNextPage

`func (o *WorkflowsListNodeExecutionsResponse) HasHasNextPage() bool`

HasHasNextPage returns a boolean if a field has been set.

### GetLastTimestamp

`func (o *WorkflowsListNodeExecutionsResponse) GetLastTimestamp() time.Time`

GetLastTimestamp returns the LastTimestamp field if non-nil, zero value otherwise.

### GetLastTimestampOk

`func (o *WorkflowsListNodeExecutionsResponse) GetLastTimestampOk() (*time.Time, bool)`

GetLastTimestampOk returns a tuple with the LastTimestamp field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastTimestamp

`func (o *WorkflowsListNodeExecutionsResponse) SetLastTimestamp(v time.Time)`

SetLastTimestamp sets LastTimestamp field to given value.

### HasLastTimestamp

`func (o *WorkflowsListNodeExecutionsResponse) HasLastTimestamp() bool`

HasLastTimestamp returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


