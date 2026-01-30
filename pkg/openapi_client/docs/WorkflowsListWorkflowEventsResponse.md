# WorkflowsListWorkflowEventsResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Events** | Pointer to [**[]WorkflowsWorkflowEventWithExecutions**](WorkflowsWorkflowEventWithExecutions.md) |  | [optional] 
**TotalCount** | Pointer to **int64** |  | [optional] 
**HasNextPage** | Pointer to **bool** |  | [optional] 
**LastTimestamp** | Pointer to **time.Time** |  | [optional] 

## Methods

### NewWorkflowsListWorkflowEventsResponse

`func NewWorkflowsListWorkflowEventsResponse() *WorkflowsListWorkflowEventsResponse`

NewWorkflowsListWorkflowEventsResponse instantiates a new WorkflowsListWorkflowEventsResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWorkflowsListWorkflowEventsResponseWithDefaults

`func NewWorkflowsListWorkflowEventsResponseWithDefaults() *WorkflowsListWorkflowEventsResponse`

NewWorkflowsListWorkflowEventsResponseWithDefaults instantiates a new WorkflowsListWorkflowEventsResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetEvents

`func (o *WorkflowsListWorkflowEventsResponse) GetEvents() []WorkflowsWorkflowEventWithExecutions`

GetEvents returns the Events field if non-nil, zero value otherwise.

### GetEventsOk

`func (o *WorkflowsListWorkflowEventsResponse) GetEventsOk() (*[]WorkflowsWorkflowEventWithExecutions, bool)`

GetEventsOk returns a tuple with the Events field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEvents

`func (o *WorkflowsListWorkflowEventsResponse) SetEvents(v []WorkflowsWorkflowEventWithExecutions)`

SetEvents sets Events field to given value.

### HasEvents

`func (o *WorkflowsListWorkflowEventsResponse) HasEvents() bool`

HasEvents returns a boolean if a field has been set.

### GetTotalCount

`func (o *WorkflowsListWorkflowEventsResponse) GetTotalCount() int64`

GetTotalCount returns the TotalCount field if non-nil, zero value otherwise.

### GetTotalCountOk

`func (o *WorkflowsListWorkflowEventsResponse) GetTotalCountOk() (*int64, bool)`

GetTotalCountOk returns a tuple with the TotalCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotalCount

`func (o *WorkflowsListWorkflowEventsResponse) SetTotalCount(v int64)`

SetTotalCount sets TotalCount field to given value.

### HasTotalCount

`func (o *WorkflowsListWorkflowEventsResponse) HasTotalCount() bool`

HasTotalCount returns a boolean if a field has been set.

### GetHasNextPage

`func (o *WorkflowsListWorkflowEventsResponse) GetHasNextPage() bool`

GetHasNextPage returns the HasNextPage field if non-nil, zero value otherwise.

### GetHasNextPageOk

`func (o *WorkflowsListWorkflowEventsResponse) GetHasNextPageOk() (*bool, bool)`

GetHasNextPageOk returns a tuple with the HasNextPage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHasNextPage

`func (o *WorkflowsListWorkflowEventsResponse) SetHasNextPage(v bool)`

SetHasNextPage sets HasNextPage field to given value.

### HasHasNextPage

`func (o *WorkflowsListWorkflowEventsResponse) HasHasNextPage() bool`

HasHasNextPage returns a boolean if a field has been set.

### GetLastTimestamp

`func (o *WorkflowsListWorkflowEventsResponse) GetLastTimestamp() time.Time`

GetLastTimestamp returns the LastTimestamp field if non-nil, zero value otherwise.

### GetLastTimestampOk

`func (o *WorkflowsListWorkflowEventsResponse) GetLastTimestampOk() (*time.Time, bool)`

GetLastTimestampOk returns a tuple with the LastTimestamp field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastTimestamp

`func (o *WorkflowsListWorkflowEventsResponse) SetLastTimestamp(v time.Time)`

SetLastTimestamp sets LastTimestamp field to given value.

### HasLastTimestamp

`func (o *WorkflowsListWorkflowEventsResponse) HasLastTimestamp() bool`

HasLastTimestamp returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


