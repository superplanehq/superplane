# SecretsDescribeSecretResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Secret** | Pointer to [**SecretsSecret**](SecretsSecret.md) |  | [optional] 

## Methods

### NewSecretsDescribeSecretResponse

`func NewSecretsDescribeSecretResponse() *SecretsDescribeSecretResponse`

NewSecretsDescribeSecretResponse instantiates a new SecretsDescribeSecretResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSecretsDescribeSecretResponseWithDefaults

`func NewSecretsDescribeSecretResponseWithDefaults() *SecretsDescribeSecretResponse`

NewSecretsDescribeSecretResponseWithDefaults instantiates a new SecretsDescribeSecretResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSecret

`func (o *SecretsDescribeSecretResponse) GetSecret() SecretsSecret`

GetSecret returns the Secret field if non-nil, zero value otherwise.

### GetSecretOk

`func (o *SecretsDescribeSecretResponse) GetSecretOk() (*SecretsSecret, bool)`

GetSecretOk returns a tuple with the Secret field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSecret

`func (o *SecretsDescribeSecretResponse) SetSecret(v SecretsSecret)`

SetSecret sets Secret field to given value.

### HasSecret

`func (o *SecretsDescribeSecretResponse) HasSecret() bool`

HasSecret returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


