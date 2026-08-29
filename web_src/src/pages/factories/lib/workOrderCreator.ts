import type {
  FactoriesAutomationRef,
  FactoriesWorkOrder,
  FactoriesWorkOrderCreator,
  SuperplaneFactoriesUserRef,
} from "@/api-client";
import type { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { getUserInitials, type OrgUserDisplay } from "@/lib/orgUserDisplay";

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
export function workOrderOwnerDisplay(order: Pick<FactoriesWorkOrder, "createdBy">, fallback: OrgUserDisplay) {
  const automation = order.createdBy?.automation;
  if (automation) {
    return buildAutomationCreatorDisplay(automation) ?? userOwnerDisplay(order.createdBy?.user, fallback);
  }
  return userOwnerDisplay(order.createdBy?.user, fallback);
}

function userOwnerDisplay(user: SuperplaneFactoriesUserRef | undefined, fallback: OrgUserDisplay): OrgUserDisplay {
  if (!user || (!user.id && !user.name)) {
    return fallback;
  }
  const name = user.name?.trim() || fallback.name;
  return {
    id: user.id ?? fallback.id,
    name,
    initials: getUserInitials(name),
    avatarUrl: user.id === fallback.id ? fallback.avatarUrl : undefined,
  };
}
