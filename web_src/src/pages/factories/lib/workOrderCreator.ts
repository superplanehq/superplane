import type { FactoriesAutomationRef, FactoriesWorkOrderCreator } from "@/api-client";
import type { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { getUserInitials, type OrgUserDisplay } from "@/lib/orgUserDisplay";

type ResolveUserFn = ReturnType<typeof useOrgUserLookup>["resolveUser"];

// Resolves a work order's `createdBy` union into a display suitable for
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
