{{- $connectionURL := .ConnectionURL }}
You are now connected to {{ $connectionURL }}
---
You can now start using the following repositories:
| Name | URL |
|------|-----|
{{- range .Repos }}
| `{{ .FullName }}` | {{ .HTMLURL }}
{{- end }}
