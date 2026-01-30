# SetupOwnerRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Email** | **string** | Owner email address | 
**FirstName** | **string** | Owner first name | 
**LastName** | **string** | Owner last name | 
**Password** | **string** | Owner password | 
**SmtpEnabled** | Pointer to **bool** | Enable SMTP email configuration | [optional] 
**SmtpHost** | Pointer to **string** | SMTP server host | [optional] 
**SmtpPort** | Pointer to **int32** | SMTP server port | [optional] 
**SmtpUsername** | Pointer to **string** | SMTP username | [optional] 
**SmtpPassword** | Pointer to **string** | SMTP password | [optional] 
**SmtpFromName** | Pointer to **string** | SMTP from name | [optional] 
**SmtpFromEmail** | Pointer to **string** | SMTP from email address | [optional] 
**SmtpUseTls** | Pointer to **bool** | Use TLS for SMTP connection | [optional] 

## Methods

### NewSetupOwnerRequest

`func NewSetupOwnerRequest(email string, firstName string, lastName string, password string, ) *SetupOwnerRequest`

NewSetupOwnerRequest instantiates a new SetupOwnerRequest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSetupOwnerRequestWithDefaults

`func NewSetupOwnerRequestWithDefaults() *SetupOwnerRequest`

NewSetupOwnerRequestWithDefaults instantiates a new SetupOwnerRequest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetEmail

`func (o *SetupOwnerRequest) GetEmail() string`

GetEmail returns the Email field if non-nil, zero value otherwise.

### GetEmailOk

`func (o *SetupOwnerRequest) GetEmailOk() (*string, bool)`

GetEmailOk returns a tuple with the Email field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEmail

`func (o *SetupOwnerRequest) SetEmail(v string)`

SetEmail sets Email field to given value.


### GetFirstName

`func (o *SetupOwnerRequest) GetFirstName() string`

GetFirstName returns the FirstName field if non-nil, zero value otherwise.

### GetFirstNameOk

`func (o *SetupOwnerRequest) GetFirstNameOk() (*string, bool)`

GetFirstNameOk returns a tuple with the FirstName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFirstName

`func (o *SetupOwnerRequest) SetFirstName(v string)`

SetFirstName sets FirstName field to given value.


### GetLastName

`func (o *SetupOwnerRequest) GetLastName() string`

GetLastName returns the LastName field if non-nil, zero value otherwise.

### GetLastNameOk

`func (o *SetupOwnerRequest) GetLastNameOk() (*string, bool)`

GetLastNameOk returns a tuple with the LastName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastName

`func (o *SetupOwnerRequest) SetLastName(v string)`

SetLastName sets LastName field to given value.


### GetPassword

`func (o *SetupOwnerRequest) GetPassword() string`

GetPassword returns the Password field if non-nil, zero value otherwise.

### GetPasswordOk

`func (o *SetupOwnerRequest) GetPasswordOk() (*string, bool)`

GetPasswordOk returns a tuple with the Password field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPassword

`func (o *SetupOwnerRequest) SetPassword(v string)`

SetPassword sets Password field to given value.


### GetSmtpEnabled

`func (o *SetupOwnerRequest) GetSmtpEnabled() bool`

GetSmtpEnabled returns the SmtpEnabled field if non-nil, zero value otherwise.

### GetSmtpEnabledOk

`func (o *SetupOwnerRequest) GetSmtpEnabledOk() (*bool, bool)`

GetSmtpEnabledOk returns a tuple with the SmtpEnabled field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSmtpEnabled

`func (o *SetupOwnerRequest) SetSmtpEnabled(v bool)`

SetSmtpEnabled sets SmtpEnabled field to given value.

### HasSmtpEnabled

`func (o *SetupOwnerRequest) HasSmtpEnabled() bool`

HasSmtpEnabled returns a boolean if a field has been set.

### GetSmtpHost

`func (o *SetupOwnerRequest) GetSmtpHost() string`

GetSmtpHost returns the SmtpHost field if non-nil, zero value otherwise.

### GetSmtpHostOk

`func (o *SetupOwnerRequest) GetSmtpHostOk() (*string, bool)`

GetSmtpHostOk returns a tuple with the SmtpHost field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSmtpHost

`func (o *SetupOwnerRequest) SetSmtpHost(v string)`

SetSmtpHost sets SmtpHost field to given value.

### HasSmtpHost

`func (o *SetupOwnerRequest) HasSmtpHost() bool`

HasSmtpHost returns a boolean if a field has been set.

### GetSmtpPort

`func (o *SetupOwnerRequest) GetSmtpPort() int32`

GetSmtpPort returns the SmtpPort field if non-nil, zero value otherwise.

### GetSmtpPortOk

`func (o *SetupOwnerRequest) GetSmtpPortOk() (*int32, bool)`

GetSmtpPortOk returns a tuple with the SmtpPort field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSmtpPort

`func (o *SetupOwnerRequest) SetSmtpPort(v int32)`

SetSmtpPort sets SmtpPort field to given value.

### HasSmtpPort

`func (o *SetupOwnerRequest) HasSmtpPort() bool`

HasSmtpPort returns a boolean if a field has been set.

### GetSmtpUsername

`func (o *SetupOwnerRequest) GetSmtpUsername() string`

GetSmtpUsername returns the SmtpUsername field if non-nil, zero value otherwise.

### GetSmtpUsernameOk

`func (o *SetupOwnerRequest) GetSmtpUsernameOk() (*string, bool)`

GetSmtpUsernameOk returns a tuple with the SmtpUsername field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSmtpUsername

`func (o *SetupOwnerRequest) SetSmtpUsername(v string)`

SetSmtpUsername sets SmtpUsername field to given value.

### HasSmtpUsername

`func (o *SetupOwnerRequest) HasSmtpUsername() bool`

HasSmtpUsername returns a boolean if a field has been set.

### GetSmtpPassword

`func (o *SetupOwnerRequest) GetSmtpPassword() string`

GetSmtpPassword returns the SmtpPassword field if non-nil, zero value otherwise.

### GetSmtpPasswordOk

`func (o *SetupOwnerRequest) GetSmtpPasswordOk() (*string, bool)`

GetSmtpPasswordOk returns a tuple with the SmtpPassword field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSmtpPassword

`func (o *SetupOwnerRequest) SetSmtpPassword(v string)`

SetSmtpPassword sets SmtpPassword field to given value.

### HasSmtpPassword

`func (o *SetupOwnerRequest) HasSmtpPassword() bool`

HasSmtpPassword returns a boolean if a field has been set.

### GetSmtpFromName

`func (o *SetupOwnerRequest) GetSmtpFromName() string`

GetSmtpFromName returns the SmtpFromName field if non-nil, zero value otherwise.

### GetSmtpFromNameOk

`func (o *SetupOwnerRequest) GetSmtpFromNameOk() (*string, bool)`

GetSmtpFromNameOk returns a tuple with the SmtpFromName field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSmtpFromName

`func (o *SetupOwnerRequest) SetSmtpFromName(v string)`

SetSmtpFromName sets SmtpFromName field to given value.

### HasSmtpFromName

`func (o *SetupOwnerRequest) HasSmtpFromName() bool`

HasSmtpFromName returns a boolean if a field has been set.

### GetSmtpFromEmail

`func (o *SetupOwnerRequest) GetSmtpFromEmail() string`

GetSmtpFromEmail returns the SmtpFromEmail field if non-nil, zero value otherwise.

### GetSmtpFromEmailOk

`func (o *SetupOwnerRequest) GetSmtpFromEmailOk() (*string, bool)`

GetSmtpFromEmailOk returns a tuple with the SmtpFromEmail field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSmtpFromEmail

`func (o *SetupOwnerRequest) SetSmtpFromEmail(v string)`

SetSmtpFromEmail sets SmtpFromEmail field to given value.

### HasSmtpFromEmail

`func (o *SetupOwnerRequest) HasSmtpFromEmail() bool`

HasSmtpFromEmail returns a boolean if a field has been set.

### GetSmtpUseTls

`func (o *SetupOwnerRequest) GetSmtpUseTls() bool`

GetSmtpUseTls returns the SmtpUseTls field if non-nil, zero value otherwise.

### GetSmtpUseTlsOk

`func (o *SetupOwnerRequest) GetSmtpUseTlsOk() (*bool, bool)`

GetSmtpUseTlsOk returns a tuple with the SmtpUseTls field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSmtpUseTls

`func (o *SetupOwnerRequest) SetSmtpUseTls(v bool)`

SetSmtpUseTls sets SmtpUseTls field to given value.

### HasSmtpUseTls

`func (o *SetupOwnerRequest) HasSmtpUseTls() bool`

HasSmtpUseTls returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


