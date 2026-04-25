# HTTP component

## Headers and organization secrets

For sensitive header values (for example `Authorization`), do not hardcode tokens in the canvas. Use an organization secret reference:

- **Form / builder fields**: set **Secret (name)** and **Secret key** to the organization secret and key. Optionally set **Prefix** (for example `Bearer ` with a trailing space) so the outgoing header is `Prefix` + secret value.
- **YAML / canvas**: you can use the same fields at the top level of each header object, or nest under `value`:

```yaml
headers:
  - name: Authorization
    value:
      secret: my-api-token
      key: token
      prefix: "Bearer "
```

Plain string `value` remains supported for non-secret headers.

Header **names** must be static (no expressions).
