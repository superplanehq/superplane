import type { FactoriesWorkOrderCreator } from "@/api-client";
import type { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import type { OrgUserDisplay } from "@/lib/orgUserDisplay";

type ResolveUserFn = ReturnType<typeof useOrgUserLookup>["resolveUser"];

// Resolves a work order's `createdBy` union into a display user, ignoring
// the automation branch (callers should render that separately when set).
export function resolveWorkOrderCreatorDisplay(
  createdBy: FactoriesWorkOrderCreator | undefined,
  resolveUser: ResolveUserFn,
): OrgUserDisplay | null {
  const user = createdBy?.user;
  return resolveUser(user?.id, user?.name);
}
