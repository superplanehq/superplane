# UsersUserSpec

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**DisplayName** | Pointer to **string** |  | [optional] 
**AccountProviders** | Pointer to [**[]UsersAccountProvider**](UsersAccountProvider.md) |  | [optional] 

## Methods

### NewUsersUserSpec

`func NewUsersUserSpec() *UsersUserSpec`

NewUsersUserSpec instantiates a new UsersUserSpec object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewUsersUserSpecWithDefaults

`func NewUsersUserSpecWithDefaults() *UsersUserSpec`

NewUsersUserSpecWithDefaults instantiates a new UsersUserSpec object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetDisplayName

`func (o *UsersUserSpec) GetDisplayName() string`

GetDisplayName returns the DisplayName field if non-nil, zero value otherwise.

### GetDisplayNameOk

`func (o *UsersUserSpec) GetDisplayNameOk() (*string, bool)`

GetDisplayNameOk returns a tuple with the DisplayName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDisplayName

`func (o *UsersUserSpec) SetDisplayName(v string)`

SetDisplayName sets DisplayName field to given value.

### HasDisplayName

`func (o *UsersUserSpec) HasDisplayName() bool`

HasDisplayName returns a boolean if a field has been set.

### GetAccountProviders

`func (o *UsersUserSpec) GetAccountProviders() []UsersAccountProvider`

GetAccountProviders returns the AccountProviders field if non-nil, zero value otherwise.

### GetAccountProvidersOk

`func (o *UsersUserSpec) GetAccountProvidersOk() (*[]UsersAccountProvider, bool)`

GetAccountProvidersOk returns a tuple with the AccountProviders field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAccountProviders

`func (o *UsersUserSpec) SetAccountProviders(v []UsersAccountProvider)`

SetAccountProviders sets AccountProviders field to given value.

### HasAccountProviders

`func (o *UsersUserSpec) HasAccountProviders() bool`

HasAccountProviders returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


