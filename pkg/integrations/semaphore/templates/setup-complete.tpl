{{- $organizationURL := .OrganizationURL }}
You are now connected to {{ $organizationURL }}
---
You can now start using the following projects:
| Project | Repository |
|---------|------------|
{{- range .Projects }}
| [{{ .Metadata.ProjectName }}]({{ $organizationURL }}/projects/{{ .Metadata.ProjectName }}) | `{{ .Spec.Repository.URL }}` |
{{- end }}