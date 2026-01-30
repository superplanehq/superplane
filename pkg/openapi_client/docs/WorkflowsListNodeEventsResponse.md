# WorkflowsListNodeEventsResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Events** | Pointer to [**[]WorkflowsWorkflowEvent**](WorkflowsWorkflowEvent.md) |  | [optional] 
**TotalCount** | Pointer to **int64** |  | [optional] 
**HasNextPage** | Pointer to **bool** |  | [optional] 
**LastTimestamp** | Pointer to **time.Time** |  | [optional] 

## Methods

### NewWorkflowsListNodeEventsResponse

`func NewWorkflowsListNodeEventsResponse() *WorkflowsListNodeEventsResponse`

NewWorkflowsListNodeEventsResponse instantiates a new WorkflowsListNodeEventsResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWorkflowsListNodeEventsResponseWithDefaults

`func NewWorkflowsListNodeEventsResponseWithDefaults() *WorkflowsListNodeEventsResponse`

NewWorkflowsListNodeEventsResponseWithDefaults instantiates a new WorkflowsListNodeEventsResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetEvents

`func (o *WorkflowsListNodeEventsResponse) GetEvents() []WorkflowsWorkflowEvent`

GetEvents returns the Events field if non-nil, zero value otherwise.

### GetEventsOk

`func (o *WorkflowsListNodeEventsResponse) GetEventsOk() (*[]WorkflowsWorkflowEvent, bool)`

GetEventsOk returns a tuple with the Events field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEvents

`func (o *WorkflowsListNodeEventsResponse) SetEvents(v []WorkflowsWorkflowEvent)`

SetEvents sets Events field to given value.

### HasEvents

`func (o *WorkflowsListNodeEventsResponse) HasEvents() bool`

HasEvents returns a boolean if a field has been set.

### GetTotalCount

`func (o *WorkflowsListNodeEventsResponse) GetTotalCount() int64`

GetTotalCount returns the TotalCount field if non-nil, zero value otherwise.

### GetTotalCountOk

`func (o *WorkflowsListNodeEventsResponse) GetTotalCountOk() (*int64, bool)`

GetTotalCountOk returns a tuple with the TotalCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotalCount

`func (o *WorkflowsListNodeEventsResponse) SetTotalCount(v int64)`

SetTotalCount sets TotalCount field to given value.

### HasTotalCount

`func (o *WorkflowsListNodeEventsResponse) HasTotalCount() bool`

HasTotalCount returns a boolean if a field has been set.

### GetHasNextPage

`func (o *WorkflowsListNodeEventsResponse) GetHasNextPage() bool`

GetHasNextPage returns the HasNextPage field if non-nil, zero value otherwise.

### GetHasNextPageOk

`func (o *WorkflowsListNodeEventsResponse) GetHasNextPageOk() (*bool, bool)`

GetHasNextPageOk returns a tuple with the HasNextPage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHasNextPage

`func (o *WorkflowsListNodeEventsResponse) SetHasNextPage(v bool)`

SetHasNextPage sets HasNextPage field to given value.

### HasHasNextPage

`func (o *WorkflowsListNodeEventsResponse) HasHasNextPage() bool`

HasHasNextPage returns a boolean if a field has been set.

### GetLastTimestamp

`func (o *WorkflowsListNodeEventsResponse) GetLastTimestamp() time.Time`

GetLastTimestamp returns the LastTimestamp field if non-nil, zero value otherwise.

### GetLastTimestampOk

`func (o *WorkflowsListNodeEventsResponse) GetLastTimestampOk() (*time.Time, bool)`

GetLastTimestampOk returns a tuple with the LastTimestamp field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastTimestamp

`func (o *WorkflowsListNodeEventsResponse) SetLastTimestamp(v time.Time)`

SetLastTimestamp sets LastTimestamp field to given value.

### HasLastTimestamp

`func (o *WorkflowsListNodeEventsResponse) HasLastTimestamp() bool`

HasLastTimestamp returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


