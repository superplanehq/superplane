- Go to https://github.com/settings/personal-access-tokens
- Find the token you want to update
- Edit the repository permissions to include the following:
| Resource | Permission Level |
|---------|------------|
{{- range $key, $value := .RepoPermissions }}
| {{ $key }} | {{ $value }} |
{{- end }}

- Edit the organization permissions to include the following:

| Resource | Permission Level |
|---------|------------|
{{- range $key, $value := .OrgPermissions }}
| {{ $key }} | {{ $value }} |
{{- end }}
