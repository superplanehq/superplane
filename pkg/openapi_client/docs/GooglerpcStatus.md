# GooglerpcStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Code** | Pointer to **int32** |  | [optional] 
**Message** | Pointer to **string** |  | [optional] 
**Details** | Pointer to [**[]ProtobufAny**](ProtobufAny.md) |  | [optional] 

## Methods

### NewGooglerpcStatus

`func NewGooglerpcStatus() *GooglerpcStatus`

NewGooglerpcStatus instantiates a new GooglerpcStatus object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewGooglerpcStatusWithDefaults

`func NewGooglerpcStatusWithDefaults() *GooglerpcStatus`

NewGooglerpcStatusWithDefaults instantiates a new GooglerpcStatus object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetCode

`func (o *GooglerpcStatus) GetCode() int32`

GetCode returns the Code field if non-nil, zero value otherwise.

### GetCodeOk

`func (o *GooglerpcStatus) GetCodeOk() (*int32, bool)`

GetCodeOk returns a tuple with the Code field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCode

`func (o *GooglerpcStatus) SetCode(v int32)`

SetCode sets Code field to given value.

### HasCode

`func (o *GooglerpcStatus) HasCode() bool`

HasCode returns a boolean if a field has been set.

### GetMessage

`func (o *GooglerpcStatus) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *GooglerpcStatus) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *GooglerpcStatus) SetMessage(v string)`

SetMessage sets Message field to given value.

### HasMessage

`func (o *GooglerpcStatus) HasMessage() bool`

HasMessage returns a boolean if a field has been set.

### GetDetails

`func (o *GooglerpcStatus) GetDetails() []ProtobufAny`

GetDetails returns the Details field if non-nil, zero value otherwise.

### GetDetailsOk

`func (o *GooglerpcStatus) GetDetailsOk() (*[]ProtobufAny, bool)`

GetDetailsOk returns a tuple with the Details field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDetails

`func (o *GooglerpcStatus) SetDetails(v []ProtobufAny)`

SetDetails sets Details field to given value.

### HasDetails

`func (o *GooglerpcStatus) HasDetails() bool`

HasDetails returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


