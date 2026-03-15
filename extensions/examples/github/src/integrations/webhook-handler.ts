import type { IntegrationWebhookHandler, RuntimeValue } from "@superplanehq/sdk";
import { githubRequest, type GitHubWebhook, parseRepositoryOwnerAndName, readGitHubToken } from "../lib/github.js";

interface WebhookConfiguration {
  eventType?: string;
  eventTypes?: string[];
  repository?: string;
}

interface ProvisionedWebhookMetadata {
  id: number;
  webhookName: string;
}

export const githubWebhookHandler = {
  compareConfig({ current, requested }) {
    const normalizedCurrent = normalizeWebhookConfiguration(current);
    const normalizedRequested = normalizeWebhookConfiguration(requested);

    if (normalizedCurrent.repository !== normalizedRequested.repository) {
      return false;
    }

    const currentEvents = normalizeEventTypes(normalizedCurrent);
    const requestedEvents = normalizeEventTypes(normalizedRequested);
    if (currentEvents.length !== requestedEvents.length) {
      return false;
    }

    const currentSet = new Set(currentEvents);
    return requestedEvents.every((event) => currentSet.has(event));
  },
  merge({ current }) {
    return {
      merged: current,
      changed: false,
    };
  },
  async setup({ runtime, webhook }) {
    const config = normalizeWebhookConfiguration(await webhook.getConfiguration());
    const token = await readGitHubToken(runtime.integration);
    const { owner, repository } = parseRepositoryOwnerAndName(config.repository ?? null);
    const secret = await webhook.getSecret();
    const events = normalizeEventTypes(config);

    const hook = await githubRequest<GitHubWebhook>(runtime.http, {
      method: "POST",
      path: `/repos/${owner}/${repository}/hooks`,
      token,
      body: {
        active: true,
        events,
        config: {
          url: await webhook.getURL(),
          secret: new TextDecoder().decode(secret),
          content_type: "json",
        },
      },
    });

    return {
      id: hook.id,
      webhookName: hook.name,
    };
  },
  async cleanup({ runtime, webhook }) {
    const config = normalizeWebhookConfiguration(await webhook.getConfiguration());
    const metadata = normalizeProvisionedWebhookMetadata(await webhook.getMetadata());
    if (!metadata) {
      return;
    }

    const token = await readGitHubToken(runtime.integration);
    const { owner, repository } = parseRepositoryOwnerAndName(config.repository ?? null);
    await githubRequest<unknown>(runtime.http, {
      method: "DELETE",
      path: `/repos/${owner}/${repository}/hooks/${metadata.id}`,
      token,
    });
  },
} satisfies IntegrationWebhookHandler;

function normalizeWebhookConfiguration(value: RuntimeValue | null | undefined): WebhookConfiguration {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return {};
  }

  return value as WebhookConfiguration;
}

function normalizeEventTypes(configuration: WebhookConfiguration): string[] {
  if (Array.isArray(configuration.eventTypes) && configuration.eventTypes.length > 0) {
    return configuration.eventTypes;
  }

  if (typeof configuration.eventType === "string" && configuration.eventType) {
    return [configuration.eventType];
  }

  return [];
}

function normalizeProvisionedWebhookMetadata(value: RuntimeValue): ProvisionedWebhookMetadata | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }

  const id = "id" in value ? value.id : undefined;
  const webhookName = "webhookName" in value ? value.webhookName : undefined;
  if (typeof id !== "number" || !Number.isInteger(id)) {
    return null;
  }

  return {
    id,
    webhookName: typeof webhookName === "string" ? webhookName : "",
  };
}
