# GraphQL component (workflow)

Native component name: `graphql`. Sends a **POST** with `Content-Type: application/json` to a GraphQL HTTP endpoint. The request body is built in the backend: `query` (string from the multi-line field) and optional `variables` (key/value map).

- **URL**: endpoint (e.g. `https://api.example.com/graphql`).

- **Query**: GraphQL document; use a multi-line string—no need to JSON-escape the document in the canvas.

- **Variables**: list of key/value; values support expressions. Omitted or empty keys are skipped. Keys become top-level keys in the JSON `variables` object (string values; convert in the query if a server expects a non-string type).

- **Headers**: e.g. `Authorization: Bearer …`.

Outputs match the HTTP component: `data.status`, `data.headers`, `data.body`. **HTTP status** (and optional success code list) defines success vs failure, same as HTTP. A response with a non-empty top-level `errors` array in the JSON body is routed to the `failure` channel, even when the HTTP status is 200.
