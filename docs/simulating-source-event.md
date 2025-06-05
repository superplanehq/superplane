You may not always want to create GitHub webhooks or Semaphore notifications to test your local environment. In those cases, you can mock the event source by using curl and openssl to create a webhook event.

```bash
export SOURCE_ID="<YOUR_SOURCE_ID>"
export SOURCE_KEY="<YOUR_SOURCE_KEY>"
export EVENT="{\"ref\":\"v1.0\",\"ref_type\":\"tag\"}"
export SIGNATURE=$(echo -n "$EVENT" | openssl dgst -sha256 -hmac "$SOURCE_KEY")

curl -X POST \
  -H "X-Hub-Signature-256: sha256=$SIGNATURE" \
  -H "Content-Type: application/json" \
  --data "$EVENT" \
  http://localhost:8000/api/v1/sources/$SOURCE_ID/github
```
