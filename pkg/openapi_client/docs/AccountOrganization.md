# AccountOrganization

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Id** | Pointer to **string** | Organization ID | [optional] 
**Name** | Pointer to **string** | Organization name | [optional] 
**Description** | Pointer to **string** | Organization description | [optional] 
**CanvasCount** | Pointer to **int64** | Number of canvases in the organization | [optional] 
**MemberCount** | Pointer to **int64** | Number of members in the organization | [optional] 

## Methods

### NewAccountOrganization

`func NewAccountOrganization() *AccountOrganization`

NewAccountOrganization instantiates a new AccountOrganization object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewAccountOrganizationWithDefaults

`func NewAccountOrganizationWithDefaults() *AccountOrganization`

NewAccountOrganizationWithDefaults instantiates a new AccountOrganization object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetId

`func (o *AccountOrganization) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *AccountOrganization) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *AccountOrganization) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *AccountOrganization) HasId() bool`

HasId returns a boolean if a field has been set.

### GetName

`func (o *AccountOrganization) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *AccountOrganization) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *AccountOrganization) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *AccountOrganization) HasName() bool`

HasName returns a boolean if a field has been set.

### GetDescription

`func (o *AccountOrganization) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *AccountOrganization) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *AccountOrganization) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *AccountOrganization) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetCanvasCount

`func (o *AccountOrganization) GetCanvasCount() int64`

GetCanvasCount returns the CanvasCount field if non-nil, zero value otherwise.

### GetCanvasCountOk

`func (o *AccountOrganization) GetCanvasCountOk() (*int64, bool)`

GetCanvasCountOk returns a tuple with the CanvasCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCanvasCount

`func (o *AccountOrganization) SetCanvasCount(v int64)`

SetCanvasCount sets CanvasCount field to given value.

### HasCanvasCount

`func (o *AccountOrganization) HasCanvasCount() bool`

HasCanvasCount returns a boolean if a field has been set.

### GetMemberCount

`func (o *AccountOrganization) GetMemberCount() int64`

GetMemberCount returns the MemberCount field if non-nil, zero value otherwise.

### GetMemberCountOk

`func (o *AccountOrganization) GetMemberCountOk() (*int64, bool)`

GetMemberCountOk returns a tuple with the MemberCount field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMemberCount

`func (o *AccountOrganization) SetMemberCount(v int64)`

SetMemberCount sets MemberCount field to given value.

### HasMemberCount

`func (o *AccountOrganization) HasMemberCount() bool`

HasMemberCount returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


