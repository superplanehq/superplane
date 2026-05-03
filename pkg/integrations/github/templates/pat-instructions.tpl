- Go to https://github.com/settings/personal-access-tokens/new
- Generate a new fine-grained personal access token
- **Token name**: `SuperPlane`
- **Resource owner**: `{{ .Owner }}`
- **Expiration**: based on your security policy
- Under **Repository access**, choose the repositories SuperPlane should access

---

### Repository permissions

Based on the capabilities you selected, these are the repository permissions you need to grant to the token:

| Resource | Permission Level |
|----------|------------------|
{{- range $key, $value := .RepoPermissions }}
| {{ $key }} | {{ $value }} |
{{- end }}

{{ if .OrgPermissions }}

---

### Organization permissions

Based on the capabilities you selected, these are the organization permissions you need to grant to the token:

| Resource | Permission Level |
|----------|------------------|
{{- range $key, $value := .OrgPermissions }}
| {{ $key }} | {{ $value }} |
{{- end }}

{{ end }}
