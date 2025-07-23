Secrets allow you to store sensitive values and share them in your canvas. You can create a secret that will be managed by SuperPlane itself with:

```yaml
kind: Secret
metadata:
  name: my-secret
  domainType: DOMAIN_TYPE_CANVAS
  domainId: canvas-123
spec:
  provider: local
  local:
    key: XXX
```

And then, to use it in your stage:

```yaml
kind: Stage
spec:
  secrets:
    - name: MY_SECRET_KEY
      valueFrom:
        secret:
          name: my-secret
          key: key

  executor:
    type: TYPE_SEMAPHORE
    integration:
      name: semaphore
    semaphore:
      project: my-semaphore-project
      branch: main
      pipelineFile: .semaphore/pipeline_3.yml
      parameters:
        MY_SECRET_KEY: ${{ secrets.MY_SECRET_KEY }}
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