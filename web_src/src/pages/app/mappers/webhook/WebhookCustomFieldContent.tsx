import { useState } from "react";
import {
  DEFAULT_WEBHOOK_HEADER_TOKEN_NAME,
  DEFAULT_WEBHOOK_SIGNATURE_HEADER,
  generateWebhookCode,
} from "@/lib/webhookRequestExample";

import { CopyCodeButton, ResetAuthButton } from "./fieldComponents";

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

export function WebhookCustomFieldContent({ nodeId, metadata, config }: WebhookCustomFieldContentProps) {
  const authMethod = config?.authentication || "none";
  const headerName = config?.headerName || DEFAULT_WEBHOOK_HEADER_TOKEN_NAME;
  const signatureHeaderName = config?.signatureHeader?.trim() || DEFAULT_WEBHOOK_SIGNATURE_HEADER;
  const webhookUrl = metadata?.url || "[URL GENERATED ONCE THE CANVAS IS SAVED]";
  const [currentSecret, setCurrentSecret] = useState<string | null>(null);

  const { title, description, code } = generateWebhookCode(authMethod, {
    webhookUrl,
    headerName,
    signatureHeaderName,
    secret: currentSecret ?? undefined,
  });

  return (
    <div className="border-t-1 border-gray-200 pt-4">
      <div className="space-y-3">
        <div>
          <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{title}</span>
          <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">{description}</p>

          <div className="mt-3">
            <label
              htmlFor="webhook-url-input"
              className="text-xs font-medium text-gray-600 dark:text-gray-400 uppercase tracking-wide"
            >
              Webhook URL
            </label>
            <div className="relative group mt-1">
              <input
                id="webhook-url-input"
                type="text"
                value={webhookUrl}
                readOnly
                className="w-full text-xs text-gray-800 dark:text-gray-100 mt-1 border-1 border-orange-950/20 px-2.5 py-2 bg-orange-50 dark:bg-amber-800 rounded-md font-mono"
              />
              <CopyCodeButton code={webhookUrl} />
            </div>
          </div>

          <div className="relative group mt-3">
            <p className="text-xs font-medium text-gray-600 dark:text-gray-400 uppercase tracking-wide">Code Example</p>
            <div className="relative group mt-1">
              <pre className="text-xs text-gray-800 dark:text-gray-100 mt-1 border-1 border-orange-950/20 px-2.5 py-2 bg-orange-50 dark:bg-amber-800 rounded-md font-mono whitespace-pre overflow-x-auto">
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
            <p className="text-sm text-gray-600 dark:text-gray-400 mt-1">
              Save the canvas to generate a webhook URL and to be able of generating authentication secrets
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
