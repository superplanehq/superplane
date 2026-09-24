import type { OrganizationsIntegration } from "@/api-client";

export type BrokenIntegrationReason = "error" | "incomplete";

export interface BrokenIntegration {
  /** Integration instance id, used to link to the detail page. */
  id: string;
  /** Instance name, e.g. "github-main". */
  name: string;
  /** Provider type name, e.g. "github", used for the icon and display name. */
  integrationName: string;
  reason: BrokenIntegrationReason;
  /** Backend-provided detail, e.g. "App was uninstalled". */
  description?: string;
  /** Next repair step shown on the action button, e.g. "Reconnect". */
  actionLabel: string;
}

const REINSTALL_HINTS = ["uninstall", "app was removed", "app removed"];
// OAuth recovery: re-run the authorization flow rather than paste a new key.
// Checked before REPLACE_KEY_HINTS so phrases like "refresh token",
// "token refresh", or "save access token" do not fall through to the generic
// "token" hint.
//
// "store access token" / "save access token" are the OAuth callback failures
// emitted by Jira ("failed to store access token"), Linear ("failed to save
// access token"), and GitLab's OAuth path - all of which need a re-authorize,
// not a new pasted key. The pasted-key path emits "access token is required"
// instead, so it still resolves to "Replace key" below.
//
// "token refresh" covers Jira's OAuth callback failure to schedule the refresh
// ("failed to schedule token refresh"), which reverses the "refresh token"
// word order and would otherwise match the generic "token" hint. DockerHub
// emits the identical string on its pasted-token path, so that phrase is only
// treated as a reconnect signal off KEY_BASED_TOKEN_REFRESH_PROVIDERS (below).
const RECONNECT_HINTS = [
  "reconnect",
  "re-connect",
  "reauthor",
  "re-author",
  "reauthenticate",
  "oauth",
  "offline_access",
  "refresh token",
  "token refresh",
  "store access token",
  "save access token",
];
const REPLACE_KEY_HINTS = ["expired", "key", "token", "credential", "secret", "unauthorized", "invalid"];

// Pasted-token providers that schedule their own access-token refresh emit the
// exact same "failed to schedule token refresh" message that Jira's OAuth
// callback does (DockerHub: pkg/integrations/dockerhub/dockerhub.go). For those
// providers the credential is a pasted key, so the "token refresh" phrase must
// not route to "Reconnect" - recovery is a new key, not a re-authorize. There
// is no distinguishing text, so the provider name is the only signal.
const KEY_BASED_TOKEN_REFRESH_PROVIDERS = new Set(["dockerhub"]);

/**
 * Chooses the next repair step from the state description text, using the
 * provider name to break ties when the text alone is ambiguous. Falls back
 * to "Reconnect" so every broken integration always offers an action.
 */
export function repairActionLabel(description: string | undefined, integrationName?: string): string {
  const text = (description ?? "").toLowerCase();
  const provider = (integrationName ?? "").toLowerCase();
  if (REINSTALL_HINTS.some((hint) => text.includes(hint))) {
    return "Reinstall app";
  }
  // "token refresh" is an OAuth reconnect signal for OAuth providers, but a
  // pasted-token provider emits the identical string when its own refresh
  // scheduling fails; keep that on the "Replace key" path below.
  const reconnectFromTokenRefreshOnly =
    text.includes("token refresh") && KEY_BASED_TOKEN_REFRESH_PROVIDERS.has(provider);
  if (!reconnectFromTokenRefreshOnly && RECONNECT_HINTS.some((hint) => text.includes(hint))) {
    return "Reconnect";
  }
  if (REPLACE_KEY_HINTS.some((hint) => text.includes(hint))) {
    return "Replace key";
  }
  return "Reconnect";
}

type IntegrationIdentity = Pick<BrokenIntegration, "id" | "name" | "integrationName">;

/** Metadata required to classify an integration. `null` when incomplete. */
function integrationIdentity(integration: OrganizationsIntegration): IntegrationIdentity | null {
  const id = integration.metadata?.id;
  const name = integration.metadata?.name;
  const integrationName = integration.metadata?.integrationName;
  if (!id || !name || !integrationName) return null;
  return { id, name, integrationName };
}

function isStuckPending(integration: OrganizationsIntegration): boolean {
  if (integration.status?.state !== "pending") return false;
  return !integration.status?.setupState?.currentStep;
}

/** Classifies one integration instance, or returns `null` when it is not broken. */
function classifyIntegration(integration: OrganizationsIntegration): BrokenIntegration | null {
  const identity = integrationIdentity(integration);
  if (!identity) return null;

  const description = integration.status?.stateDescription;

  if (integration.status?.state === "error") {
    return {
      ...identity,
      reason: "error",
      description: description || "Connection is broken.",
      actionLabel: repairActionLabel(description, identity.integrationName),
    };
  }

  if (isStuckPending(integration)) {
    return {
      ...identity,
      reason: "incomplete",
      description: description || "Setup is not finished.",
      actionLabel: "Finish setup",
    };
  }

  return null;
}

/**
 * Finds organization integrations that need attention: integrations in the
 * `error` state, and integrations stuck `pending` outside of an active setup
 * wizard (no current setup step to resume). Ready integrations, and
 * integrations still mid-setup, are not broken.
 */
export function findBrokenIntegrations(integrations: OrganizationsIntegration[]): BrokenIntegration[] {
  return integrations
    .map((integration) => classifyIntegration(integration))
    .filter((integration): integration is BrokenIntegration => integration !== null);
}
