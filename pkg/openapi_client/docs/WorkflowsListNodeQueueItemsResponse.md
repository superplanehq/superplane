# WorkflowsListNodeQueueItemsResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Items** | Pointer to [**[]WorkflowsWorkflowNodeQueueItem**](WorkflowsWorkflowNodeQueueItem.md) |  | [optional] 
**TotalCount** | Pointer to **int64** |  | [optional] 
**HasNextPage** | Pointer to **bool** |  | [optional] 
**LastTimestamp** | Pointer to **time.Time** |  | [optional] 

## Methods

### NewWorkflowsListNodeQueueItemsResponse

`func NewWorkflowsListNodeQueueItemsResponse() *WorkflowsListNodeQueueItemsResponse`

NewWorkflowsListNodeQueueItemsResponse instantiates a new WorkflowsListNodeQueueItemsResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewWorkflowsListNodeQueueItemsResponseWithDefaults

`func NewWorkflowsListNodeQueueItemsResponseWithDefaults() *WorkflowsListNodeQueueItemsResponse`

NewWorkflowsListNodeQueueItemsResponseWithDefaults instantiates a new WorkflowsListNodeQueueItemsResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetItems

`func (o *WorkflowsListNodeQueueItemsResponse) GetItems() []WorkflowsWorkflowNodeQueueItem`

GetItems returns the Items field if non-nil, zero value otherwise.

### GetItemsOk

`func (o *WorkflowsListNodeQueueItemsResponse) GetItemsOk() (*[]WorkflowsWorkflowNodeQueueItem, bool)`

GetItemsOk returns a tuple with the Items field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetItems

`func (o *WorkflowsListNodeQueueItemsResponse) SetItems(v []WorkflowsWorkflowNodeQueueItem)`

SetItems sets Items field to given value.

### HasItems

`func (o *WorkflowsListNodeQueueItemsResponse) HasItems() bool`

HasItems returns a boolean if a field has been set.

### GetTotalCount

`func (o *WorkflowsListNodeQueueItemsResponse) GetTotalCount() int64`

GetTotalCount returns the TotalCount field if non-nil, zero value otherwise.

### GetTotalCountOk

`func (o *WorkflowsListNodeQueueItemsResponse) GetTotalCountOk() (*int64, bool)`

GetTotalCountOk returns a tuple with the TotalCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotalCount

`func (o *WorkflowsListNodeQueueItemsResponse) SetTotalCount(v int64)`

SetTotalCount sets TotalCount field to given value.

### HasTotalCount

`func (o *WorkflowsListNodeQueueItemsResponse) HasTotalCount() bool`

HasTotalCount returns a boolean if a field has been set.

### GetHasNextPage

`func (o *WorkflowsListNodeQueueItemsResponse) GetHasNextPage() bool`

GetHasNextPage returns the HasNextPage field if non-nil, zero value otherwise.

### GetHasNextPageOk

`func (o *WorkflowsListNodeQueueItemsResponse) GetHasNextPageOk() (*bool, bool)`

GetHasNextPageOk returns a tuple with the HasNextPage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetHasNextPage

`func (o *WorkflowsListNodeQueueItemsResponse) SetHasNextPage(v bool)`

SetHasNextPage sets HasNextPage field to given value.

### HasHasNextPage

`func (o *WorkflowsListNodeQueueItemsResponse) HasHasNextPage() bool`

HasHasNextPage returns a boolean if a field has been set.

### GetLastTimestamp

`func (o *WorkflowsListNodeQueueItemsResponse) GetLastTimestamp() time.Time`

GetLastTimestamp returns the LastTimestamp field if non-nil, zero value otherwise.

### GetLastTimestampOk

`func (o *WorkflowsListNodeQueueItemsResponse) GetLastTimestampOk() (*time.Time, bool)`

GetLastTimestampOk returns a tuple with the LastTimestamp field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastTimestamp

`func (o *WorkflowsListNodeQueueItemsResponse) SetLastTimestamp(v time.Time)`

SetLastTimestamp sets LastTimestamp field to given value.

### HasLastTimestamp

`func (o *WorkflowsListNodeQueueItemsResponse) HasLastTimestamp() bool`

HasLastTimestamp returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


