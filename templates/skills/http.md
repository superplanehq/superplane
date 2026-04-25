# HTTP Request component

## Secure API authentication

Use the **Authorization** section (not the generic **Headers** list) to send `Authorization` from an organization secret.

- **Credential**: pick the organization secret and the key name that stores the token.
- **Value prefix**: defaults to `Bearer ` (include the trailing space). Use an empty prefix if the secret is the full header value with no scheme.

The **Headers** list remains for non-sensitive headers; if both define `Authorization`, the **Authorization** section wins at runtime.

## Typical REST call

- **Method** / **URL** as needed.
- Optional **Body** (JSON, form, text, or XML) for POST/PUT/PATCH.
