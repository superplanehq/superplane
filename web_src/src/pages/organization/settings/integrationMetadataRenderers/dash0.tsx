import { IntegrationMetadataRenderer } from "./types";
import { CopyButton } from "./components";

export const dash0MetadataRenderer: IntegrationMetadataRenderer = ({ integration }) => {
  const metadata = integration.status?.metadata as Record<string, unknown> | undefined;
  const webhookUrl = metadata?.webhookUrl;
  if (typeof webhookUrl !== "string" || !webhookUrl.trim()) {
    return null;
  }

  const normalizedWebhookURL = webhookUrl.trim();

  return (
    <div className="rounded-md border border-blue-200 bg-blue-50 p-4 text-sm text-blue-900">
      <div className="mb-3 font-medium">Dash0 Notification Webhook</div>
      <div className="mb-2 text-xs text-blue-900/90">Create a notification channel in Dash0:</div>
      <ol className="mb-3 list-decimal space-y-1 pl-4 text-xs text-blue-900/90">
        <li>
          Go to{" "}
          <a
            href="https://app.dash0.com/settings/notifications"
            target="_blank"
            rel="noopener noreferrer"
            className="font-semibold [text-decoration:underline!important] [text-underline-offset:2px] [text-decoration-thickness:2px]"
          >
            Organization Settings &gt; Notification Channels
          </a>
          .
        </li>
        <li>Add a new notification channel.</li>
        <li>Copy the webhook URL below and paste it in the &quot;Webhook URL&quot; field.</li>
      </ol>
      <div className="mb-2 text-xs text-blue-900/90">Use this webhook URL as the destination:</div>
      <div className="flex items-center gap-2">
        <div className="min-w-0 flex-1 rounded border border-blue-200 bg-white px-2.5 py-2">
          <code
            className="block flex-1 overflow-hidden text-ellipsis whitespace-nowrap font-mono text-xs leading-5 text-blue-900"
            title={normalizedWebhookURL}
          >
            {normalizedWebhookURL}
          </code>
        </div>
        <CopyButton text={normalizedWebhookURL} label="Webhook URL" />
      </div>
    </div>
  );
};
