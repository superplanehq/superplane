- Go to https://github.com/settings/personal-access-tokens/new
- Generate a new fine-grained personal access token
- **Token name**: `SuperPlane`
- **Resource owner**: `{{ .Owner }}`
- **Expiration**: based on your security policy
- Under **Repository access**, choose the repositories SuperPlane should access
- Based on the capabilities you selected, these are the permissions you need to grant to the token:

| Name | Scope | Access |
|----------|-------|------------------|
{{- range $permission := .Permissions }}
| {{ $permission.Name }} | {{ $permission.Scope }} | {{ $permission.Access }} |
{{- end }}
