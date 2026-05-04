- Go to {{ .AppURL }}
- Edit the permissions to include the following:

| Permission | Scope | Access |
|----------|-------|------------------|
{{- range $permission := .Permissions }}
| {{ $permission.Name }} | {{ $permission.Scope }} | {{ $permission.Access }} |
{{- end }}
