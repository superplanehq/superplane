import type {
  FactoriesAutomationRef,
  FactoriesWorkOrder,
  FactoriesWorkOrderCreator,
  SuperplaneFactoriesUserRef,
} from "@/api-client";
import type { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { getUserInitials, type OrgUserDisplay, type OrgUserDisplayLookup } from "@/lib/orgUserDisplay";

type ResolveUserFn = ReturnType<typeof useOrgUserLookup>["resolveUser"];

// Resolves a task's `createdBy` union into a display suitable for
// avatar/name rendering. When the creator is an automation, callers that
// want a richer rendering (e.g. a link to the app) should check
// `createdBy.automation` themselves; this helper guarantees at least a
// meaningful avatar + label instead of an "unknown" fallback.
export function resolveWorkOrderCreatorDisplay(
  createdBy: FactoriesWorkOrderCreator | undefined,
  resolveUser: ResolveUserFn,
): OrgUserDisplay | null {
  const user = createdBy?.user;
  if (user?.id || user?.name) {
    return resolveUser(user.id, user.name);
  }
  const automation = createdBy?.automation;
  if (automation) {
    return buildAutomationCreatorDisplay(automation);
  }
  return null;
}

/**
 * Plain-text creator label for a pull request: who created the task the
 * pull request belongs to, as tracked in SuperPlane. Unlike
 * `resolveWorkOrderCreatorDisplay`, this does not need the org member list
 * (no avatar), so callers that only have the raw payload can still show a
 * name.
 */
export function pullRequestCreatorLabel(createdBy: FactoriesWorkOrderCreator | undefined): string | null {
  const automation = createdBy?.automation;
  if (automation) {
    return automation.nodeName?.trim() || automation.appName?.trim() || null;
  }
  return createdBy?.user?.name?.trim() || null;
}

function buildAutomationCreatorDisplay(automation: FactoriesAutomationRef): OrgUserDisplay | null {
  const name = automation.nodeName?.trim() || automation.appName?.trim();
  if (!name) {
    return null;
  }
  const id = automation.nodeId || automation.appId || `automation:${name}`;
  return {
    id,
    name,
    initials: getUserInitials(name) || "A",
  };
}

/** Storybook/fixture owner chip for a task. Prefers automation, then user. */
export function workOrderOwnerDisplay(
  order: Pick<FactoriesWorkOrder, "createdBy">,
  fallback: OrgUserDisplay,
  resolveUser?: OrgUserDisplayLookup,
) {
  const automation = order.createdBy?.automation;
  if (automation) {
    return buildAutomationCreatorDisplay(automation) ?? userOwnerDisplay(order.createdBy?.user, fallback, resolveUser);
  }
  return userOwnerDisplay(order.createdBy?.user, fallback, resolveUser);
}

function userOwnerDisplay(
  user: SuperplaneFactoriesUserRef | undefined,
  fallback: OrgUserDisplay,
  resolveUser?: OrgUserDisplayLookup,
): OrgUserDisplay {
  if (!user || (!user.id && !user.name)) {
    return fallback;
  }
  // `resolveUser` looks the owner up against the org members list, which
  // carries the avatar image. Without it we can only show initials (or the
  // fallback's avatar when the ids happen to match).
  const resolved = resolveUser?.(user.id, user.name);
  if (resolved) {
    return resolved;
  }
  const name = user.name?.trim() || fallback.name;
  return {
    id: user.id ?? fallback.id,
    name,
    initials: getUserInitials(name),
    avatarUrl: user.id === fallback.id ? fallback.avatarUrl : undefined,
  };
}
