Update your OpenAI API key permissions before enabling the new capabilities.

If you are using OpenAI, edit the key and keep **Permissions** set to **Restricted**. Add these endpoint permissions:

| Endpoint | Access |
|----------|--------|
{{- range $permission := .Permissions }}
| `{{ $permission.Endpoint }}` | {{ $permission.Access }} |
{{- end }}

If you created a replacement key, paste it below. Otherwise, leave the field empty and submit after updating the existing key.

