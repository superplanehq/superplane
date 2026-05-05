- Go to https://github.com/settings/personal-access-tokens
- Find the token with access to {{ .Owner }} for update
- Edit the permissions to include the following:

| Permission | Scope | Access |
|----------|-------|------------------|
{{- range $permission := .Permissions }}
| {{ $permission.Name }} | {{ $permission.Scope }} | {{ $permission.Access }} |
{{- end }}
