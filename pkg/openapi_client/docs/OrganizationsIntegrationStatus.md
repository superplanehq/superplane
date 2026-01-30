# OrganizationsIntegrationStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**State** | Pointer to **string** |  | [optional] 
**StateDescription** | Pointer to **string** |  | [optional] 
**Metadata** | Pointer to **map[string]interface{}** |  | [optional] 
**BrowserAction** | Pointer to [**OrganizationsBrowserAction**](OrganizationsBrowserAction.md) |  | [optional] 
**UsedIn** | Pointer to [**[]IntegrationNodeRef**](IntegrationNodeRef.md) |  | [optional] 

## Methods

### NewOrganizationsIntegrationStatus

`func NewOrganizationsIntegrationStatus() *OrganizationsIntegrationStatus`

NewOrganizationsIntegrationStatus instantiates a new OrganizationsIntegrationStatus object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewOrganizationsIntegrationStatusWithDefaults

`func NewOrganizationsIntegrationStatusWithDefaults() *OrganizationsIntegrationStatus`

NewOrganizationsIntegrationStatusWithDefaults instantiates a new OrganizationsIntegrationStatus object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetState

`func (o *OrganizationsIntegrationStatus) GetState() string`

GetState returns the State field if non-nil, zero value otherwise.

### GetStateOk

`func (o *OrganizationsIntegrationStatus) GetStateOk() (*string, bool)`

GetStateOk returns a tuple with the State field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetState

`func (o *OrganizationsIntegrationStatus) SetState(v string)`

SetState sets State field to given value.

### HasState

`func (o *OrganizationsIntegrationStatus) HasState() bool`

HasState returns a boolean if a field has been set.

### GetStateDescription

`func (o *OrganizationsIntegrationStatus) GetStateDescription() string`

GetStateDescription returns the StateDescription field if non-nil, zero value otherwise.

### GetStateDescriptionOk

`func (o *OrganizationsIntegrationStatus) GetStateDescriptionOk() (*string, bool)`

GetStateDescriptionOk returns a tuple with the StateDescription field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetStateDescription

`func (o *OrganizationsIntegrationStatus) SetStateDescription(v string)`

SetStateDescription sets StateDescription field to given value.

### HasStateDescription

`func (o *OrganizationsIntegrationStatus) HasStateDescription() bool`

HasStateDescription returns a boolean if a field has been set.

### GetMetadata

`func (o *OrganizationsIntegrationStatus) GetMetadata() map[string]interface{}`

GetMetadata returns the Metadata field if non-nil, zero value otherwise.

### GetMetadataOk

`func (o *OrganizationsIntegrationStatus) GetMetadataOk() (*map[string]interface{}, bool)`

GetMetadataOk returns a tuple with the Metadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMetadata

`func (o *OrganizationsIntegrationStatus) SetMetadata(v map[string]interface{})`

SetMetadata sets Metadata field to given value.

### HasMetadata

`func (o *OrganizationsIntegrationStatus) HasMetadata() bool`

HasMetadata returns a boolean if a field has been set.

### GetBrowserAction

`func (o *OrganizationsIntegrationStatus) GetBrowserAction() OrganizationsBrowserAction`

GetBrowserAction returns the BrowserAction field if non-nil, zero value otherwise.

### GetBrowserActionOk

`func (o *OrganizationsIntegrationStatus) GetBrowserActionOk() (*OrganizationsBrowserAction, bool)`

GetBrowserActionOk returns a tuple with the BrowserAction field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBrowserAction

`func (o *OrganizationsIntegrationStatus) SetBrowserAction(v OrganizationsBrowserAction)`

SetBrowserAction sets BrowserAction field to given value.

### HasBrowserAction

`func (o *OrganizationsIntegrationStatus) HasBrowserAction() bool`

HasBrowserAction returns a boolean if a field has been set.

### GetUsedIn

`func (o *OrganizationsIntegrationStatus) GetUsedIn() []IntegrationNodeRef`

GetUsedIn returns the UsedIn field if non-nil, zero value otherwise.

### GetUsedInOk

`func (o *OrganizationsIntegrationStatus) GetUsedInOk() (*[]IntegrationNodeRef, bool)`

GetUsedInOk returns a tuple with the UsedIn field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUsedIn

`func (o *OrganizationsIntegrationStatus) SetUsedIn(v []IntegrationNodeRef)`

SetUsedIn sets UsedIn field to given value.

### HasUsedIn

`func (o *OrganizationsIntegrationStatus) HasUsedIn() bool`

HasUsedIn returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


