export const DEFAULT_WEBHOOK_HEADER_TOKEN_NAME = "X-Webhook-Token";
export const DEFAULT_WEBHOOK_SIGNATURE_HEADER = "X-Signature-256";

export function generateWebhookCode(
  authMethod: string,
  options: {
    webhookUrl: string;
    headerName: string;
    signatureHeaderName: string;
    secret?: string;
  },
): { title: string; description: string; code: string } {
  const { webhookUrl, headerName, signatureHeaderName, secret } = options;
  let description: string;
  let code: string;
  let title: string;
  let signatureKey: string;

  switch (authMethod) {
    case "signature":
      title = "HMAC Signature Authentication";
      description = "Use HMAC SHA-256 signature to authenticate your webhook requests.";
      signatureKey = secret || "<your-signature-key>";
      code = `export SIGNATURE_KEY="${signatureKey}"
export PAYLOAD='{"hello":"world"}'

export SIGNATURE=$(echo -n "$PAYLOAD" \\
  | openssl dgst -sha256 -hmac "$SIGNATURE_KEY" -binary \\
  | xxd -p -c 256)

curl -X POST \\
  -H "${signatureHeaderName}: sha256=$SIGNATURE" \\
  -H "Content-Type: application/json" \\
  --data-binary "$PAYLOAD" \\
  ${webhookUrl}`;
      break;

    case "bearer":
      title = "Bearer Token Authentication";
      description = "Use bearer token to authenticate your webhook requests.";
      signatureKey = secret || "<your-bearer-token>";
      code = `export BEARER_TOKEN="${signatureKey}"
export PAYLOAD='{"hello":"world"}'

curl -X POST \\
  -H "Authorization: Bearer $BEARER_TOKEN" \\
  -H "Content-Type: application/json" \\
  --data "$PAYLOAD" \\
  ${webhookUrl}`;
      break;

    case "header_token":
      title = "Header Token Authentication";
      description = `Use a raw token in the ${headerName} header to authenticate webhook requests.`;
      signatureKey = secret || "<your-header-token>";
      code = `export HEADER_TOKEN="${signatureKey}"
export PAYLOAD='{"hello":"world"}'

curl -X POST \\
  -H "${headerName}: $HEADER_TOKEN" \\
  -H "Content-Type: application/json" \\
  --data "$PAYLOAD" \\
  ${webhookUrl}`;
      break;

    default:
      title = "No Authentication";
      description = "Send webhook requests without authentication.";
      code = `export PAYLOAD='{"hello":"world"}'

curl -X POST \\
  -H "Content-Type: application/json" \\
  --data "$PAYLOAD" \\
  ${webhookUrl}`;
      break;
  }

  return { title, description, code };
}
