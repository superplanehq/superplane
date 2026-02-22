import { IntegrationMetadataRenderer } from "./types";
import { URLField } from "./components";

export const linearMetadataRenderer: IntegrationMetadataRenderer = ({ integration }) => {
  const metadata = integration.status?.metadata as Record<string, unknown> | undefined;
  const webhookUrl = metadata?.webhookUrl;
  const callbackUrl = metadata?.callbackUrl;

  const hasWebhook = typeof webhookUrl === "string" && webhookUrl.trim().length > 0;
  const hasCallback = typeof callbackUrl === "string" && callbackUrl.trim().length > 0;

  if (!hasWebhook && !hasCallback) {
    return null;
  }

  return (
    <div className="rounded-md border border-blue-200 bg-blue-50 p-4 text-sm text-blue-900">
      <div className="mb-3 font-medium">Linear OAuth Application URLs</div>
      <div className="mb-3 text-xs text-blue-900/90">
        Use these URLs when creating your OAuth2 application in{" "}
        <a
          href="https://linear.app/settings/api/applications/new"
          target="_blank"
          rel="noopener noreferrer"
          className="font-semibold [text-decoration:underline!important] [text-underline-offset:2px] [text-decoration-thickness:2px]"
        >
          Linear API Settings
        </a>
        .
      </div>
      <div className="space-y-3">
        {hasCallback && <URLField label="Callback URL" url={(callbackUrl as string).trim()} />}
        {hasWebhook && <URLField label="Webhook URL" url={(webhookUrl as string).trim()} />}
      </div>
    </div>
  );
};
