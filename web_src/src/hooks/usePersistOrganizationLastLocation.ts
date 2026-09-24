import { useEffect } from "react";

import { recordLastVisitedOrganization } from "@/lib/lastVisitedOrganization";

import { useMe } from "./useMe";
import { useRecordLastLocation } from "./useRecordLastLocation";

function canRecordOrganizationLocation(args: {
  resolvedSlug: string;
  isReserved: boolean;
  uidRedirectPath: string | null;
  meReady: boolean;
  permissionCount: number;
}): boolean {
  return Boolean(
    args.resolvedSlug && !args.isReserved && !args.uidRedirectPath && args.meReady && args.permissionCount > 0,
  );
}

/** Saves the current org screen only after /me returns permissions. Empty RBAC must not persist. */
export function usePersistOrganizationLastLocation(args: {
  accountId: string | undefined;
  resolvedSlug: string;
  isReserved: boolean;
  uidRedirectPath: string | null;
  path: string;
}): void {
  const { data: me, isSuccess: meReady } = useMe(true, args.resolvedSlug || null);
  const canRecord = canRecordOrganizationLocation({
    resolvedSlug: args.resolvedSlug,
    isReserved: args.isReserved,
    uidRedirectPath: args.uidRedirectPath,
    meReady,
    permissionCount: me?.permissions?.length ?? 0,
  });

  useEffect(() => {
    if (!args.accountId || !canRecord || !args.resolvedSlug) {
      return;
    }
    recordLastVisitedOrganization(args.accountId, args.resolvedSlug);
  }, [args.accountId, args.resolvedSlug, canRecord]);

  useRecordLastLocation(canRecord ? args.resolvedSlug : null, args.accountId, args.path);
}
