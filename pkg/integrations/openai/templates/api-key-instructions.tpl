Use an API key to let SuperPlane call the OpenAI API.

{{ if .IsDefaultBaseURL }}
If you are using OpenAI:
- Go to https://platform.openai.com/api-keys
- Create a new secret key
- Set **Permissions** to **Restricted**
- Grant these endpoint permissions:

| Endpoint | Access |
|----------|--------|
{{- range $permission := .Permissions }}
| `{{ $permission.Endpoint }}` | {{ $permission.Access }} |
{{- end }}

Do not use **Read Only** for these capabilities. It can list models, but it cannot create responses.
{{- else }}
You changed the Base URL in the previous step. Paste the API key or token for that provider.

Make sure the provider allows equivalent access for:

| Endpoint | Access |
|----------|--------|
{{- range $permission := .Permissions }}
| `{{ $permission.Endpoint }}` | {{ $permission.Access }} |
{{- end }}
{{- end }}
