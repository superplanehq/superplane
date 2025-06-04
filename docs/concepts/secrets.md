Secrets allow you to store sensitive values and share them in your canvas. You can create a secret that will be managed by SuperPlane itself with:

```yaml
kind: Secret
metadata:
  name: semaphore-access
spec:
  provider: local
  local:
    api-token: XXX
```

And then, to use it in your stage:

```yaml
kind: Stage
spec:
  secrets:
    - name: API_TOKEN
      valueFrom:
        secret:
          name: semaphore-access
          key: api-token

  executor:
    type: TYPE_SEMAPHORE
    semaphore:
      organizationUrl: https://myorg.semaphoreci.com
      apiToken: ${{ secrets.API_TOKEN }}
      projectId: dfafcfe4-cf55-4cb9-abde-c073733c9b83
      taskId: fd67cfb1-e06c-4896-a517-c648f878330a
      branch: main
      pipelineFile: .semaphore/pipeline_3.yml
      parameters: {}
```

### Other secret providers

NOTE: to be implemented once SuperPlane is an OIDC provider.

#### Vault

```yaml
#
# Secret is stored in Vault,
# We use OIDC tokens issued by Superplane to authenticate with Vault and fetch that value.
#
provider: vault
vault:
  secretName: myapp/prod/db-credentials
  region: us-east-1
  auth:
    method: oidc
    role: my-app-role

    # mount path => /v1/auth/{mountPath}/login vault login URL
    # Since 'jwt' is the default one for jwt auth in vault, we default it here too.
    # but this should be configurable.
    # See: https://developer.hashicorp.com/vault/docs/auth/jwt#jwt-authentication
    mountPath: jwt
```

#### AWS secret manager

```yaml
#
# Secret is stored in AWS secret manager,
# we just load it from there, using OIDC tokens issued by Superplane.
#
provider: aws
aws:
  secretName: myapp/prod/db-credentials
  region: us-east-1
  auth:
    method: oidc
    roleArn: arn:aws:iam::123456789012:role/MyAppAccessRole
```