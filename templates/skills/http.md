# HTTP Request

Use **Authorization** (org secret + key) for `Authorization` tokens, not the **Headers** list. Default prefix is `Bearer `; use an empty prefix for a raw token. If both set `Authorization`, **Authorization** wins.

For POST/PUT/PATCH, set **Method**, **URL**, and optional **Body** as needed.
