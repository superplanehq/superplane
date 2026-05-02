- Go to https://github.com/settings/personal-access-tokens
- Find the token you want to update
- Edit the permissions to include the following:
| Resource | Permission Level |
|---------|------------|
{{- range $key, $value := .Permissions }}
| {{ $key }} | {{ $value }} |
{{- end }}
