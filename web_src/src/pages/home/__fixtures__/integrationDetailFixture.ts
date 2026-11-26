import type { OrganizationsIntegration } from "@/api-client";

/** Serves integration details and completes webhook retries in interactive stories. */
export function matchIntegrationDetailFixture(url: URL, method: string, integrations: OrganizationsIntegration[]) {
  const detailMatch = /^\/api\/v1\/organizations\/[^/]+\/integrations\/([^/]+)$/.exec(url.pathname);
  if (detailMatch && method === "GET") {
    const integration = integrations.find((item) => item.metadata?.id === detailMatch[1]);
    return { json: integration ? { integration } : {} };
  }

  const retryMatch = /^\/api\/v1\/organizations\/[^/]+\/integrations\/([^/]+)\/webhooks\/([^/]+)\/retry$/.exec(
    url.pathname,
  );
  if (!retryMatch || method !== "POST") return null;
  const integration = integrations.find((item) => item.metadata?.id === retryMatch[1]);
  const failure = integration?.status?.webhookFailures?.find((item) => item.id === retryMatch[2]);
  if (!failure || !integration?.status) return { json: {}, status: 404 };
  integration.status.webhookFailures = integration.status.webhookFailures?.filter((item) => item.id !== failure.id);
  return { json: {} };
}
