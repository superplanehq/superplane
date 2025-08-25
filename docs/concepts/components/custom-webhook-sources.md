When you create a custom webhook event source, you are responsible for pushing the events for it to Superplane, using the URL and signature key provided by Superplane. The event should be a JSON object, and the signature should be computed using the HMAC-SHA256 algorithm.

For example, this is how you can push an event for a custom webhook event source using curl and openssl:

```bash
export SOURCE_ID="<YOUR_SOURCE_ID>"
export SOURCE_KEY="<YOUR_SOURCE_KEY>"
export EVENT="{\"version\":\"v1.0\",\"app\":\"core\"}"
export SIGNATURE=$(echo -n "$EVENT" | openssl dgst -sha256 -hmac "$SOURCE_KEY" | awk '{print $2}')

curl -X POST \
  -H "X-Signature-256: sha256=$SIGNATURE" \
  -H "Content-Type: application/json" \
  --data "$EVENT" \
  http://localhost:8000/api/v1/sources/$SOURCE_ID
```
