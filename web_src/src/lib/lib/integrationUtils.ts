/**
 * Extracts the webhook URL from an integration's status metadata, if present.
 */
export function getIntegrationWebhookUrl(metadata: { [key: string]: unknown } | undefined): string | undefined {
  if (!metadata || typeof metadata !== "object" || !("webhookUrl" in metadata)) {
    return undefined;
  }
  const url = (metadata as { webhookUrl?: string }).webhookUrl;
  return typeof url === "string" ? url : undefined;
}
