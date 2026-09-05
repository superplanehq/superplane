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
// Checked before REPLACE_KEY_HINTS so "refresh token" does not match "token".
const RECONNECT_HINTS = ["reconnect", "re-connect", "reauthor", "re-author", "reauthenticate", "oauth", "offline_access", "refresh token"];
const REPLACE_KEY_HINTS = ["expired", "key", "token", "credential", "secret", "unauthorized", "invalid"];

/**
 * Chooses the next repair step from the state description text. Falls back
 * to "Reconnect" so every broken integration always offers an action.
 */
export function repairActionLabel(description: string | undefined): string {
  const text = (description ?? "").toLowerCase();
  if (REINSTALL_HINTS.some((hint) => text.includes(hint))) {
    return "Reinstall app";
  }
  if (RECONNECT_HINTS.some((hint) => text.includes(hint))) {
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
      actionLabel: repairActionLabel(description),
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
