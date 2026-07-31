import { useState } from "react";
import { CopyCodeButton, ResetAuthButton } from "./fieldComponents";

const DEFAULT_HEADER_TOKEN_NAME = "X-Webhook-Token";
const DEFAULT_SIGNATURE_HEADER = "X-Signature-256";

interface WebhookConfiguration {
  authentication?: string;
  headerName?: string;
  signatureHeader?: string;
}

interface WebhookMetadata {
  url?: string;
  authentication?: string;
}

interface WebhookCustomFieldContentProps {
  nodeId: string;
  metadata?: WebhookMetadata;
  config?: WebhookConfiguration;
}

function generateWebhookCode(
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

export function WebhookCustomFieldContent({ nodeId, metadata, config }: WebhookCustomFieldContentProps) {
  const authMethod = config?.authentication || "none";
  const headerName = config?.headerName || DEFAULT_HEADER_TOKEN_NAME;
  const signatureHeaderName = config?.signatureHeader?.trim() || DEFAULT_SIGNATURE_HEADER;
  const webhookUrl = metadata?.url || "[URL GENERATED ONCE THE CANVAS IS SAVED]";
  const [currentSecret, setCurrentSecret] = useState<string | null>(null);

  const { title, description, code } = generateWebhookCode(authMethod, {
    webhookUrl,
    headerName,
    signatureHeaderName,
    secret: currentSecret ?? undefined,
  });

  return (
    <div className="border-t-1 border-edge-default pt-4">
      <div className="space-y-3">
        <div>
          <span className="text-sm font-medium text-content-secondary">{title}</span>
          <p className="mt-1 text-sm text-content-secondary">{description}</p>

          <div className="mt-3">
            <label
              htmlFor="webhook-url-input"
              className="text-xs font-medium text-content-secondary uppercase tracking-wide"
            >
              Webhook URL
            </label>
            <div className="relative group mt-1">
              <input
                id="webhook-url-input"
                type="text"
                value={webhookUrl}
                readOnly
                className="mt-1 w-full rounded-md border-1 border-orange-950/20 bg-orange-50 px-2.5 py-2 font-mono text-xs text-content-primary dark:bg-amber-800"
              />
              <CopyCodeButton code={webhookUrl} />
            </div>
          </div>

          <div className="relative group mt-3">
            <p className="text-xs font-medium text-content-secondary uppercase tracking-wide">Code Example</p>
            <div className="relative group mt-1">
              <pre className="mt-1 overflow-x-auto rounded-md border-1 border-orange-950/20 bg-orange-50 px-2.5 py-2 font-mono text-xs text-content-primary whitespace-pre dark:bg-amber-800">
                {code}
              </pre>
              <CopyCodeButton code={code} />
            </div>
          </div>
          {metadata?.url ? (
            <ResetAuthButton
              nodeId={nodeId}
              authMethod={authMethod}
              onSuccess={(newSecret) => {
                setCurrentSecret(newSecret);
                setTimeout(() => setCurrentSecret(null), 30000);
              }}
            />
          ) : (
            <p className="mt-1 text-sm text-content-secondary">
              Save the canvas to generate a webhook URL and to be able of generating authentication secrets
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
