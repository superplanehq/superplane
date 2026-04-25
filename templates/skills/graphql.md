# GraphQL component (workflow)

Native component name: `graphql`. Sends a **POST** with `Content-Type: application/json` to a GraphQL HTTP endpoint. The request body is built in the backend: `query` (string from the multi-line field) and optional `variables` (key/value map).

- **URL**: endpoint (e.g. `https://api.example.com/graphql`).

- **Query**: GraphQL document; use a multi-line string - no need to JSON-escape the document in the canvas.

- **Variables**: list of key/value; values support expressions. Omitted or empty keys are skipped. Keys become top-level keys in the JSON `variables` object.

- **Headers**: optional request headers.

- **Authorization**: optional bearer token from an organization Secret. When configured, SuperPlane sends `Authorization: Bearer <token>`.

Outputs match the HTTP component: `data.status`, `data.headers`, `data.body`. A response with a non-empty top-level `errors` array in the JSON body is routed to the `failure` channel, even when the HTTP status is 200.
