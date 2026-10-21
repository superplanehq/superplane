import { useEffect } from "react";

import type { OrganizationsOrganization } from "@/api-client";
import { identifyOrganizationGroup } from "@/posthog";

type RefreshOrganization = () => Promise<{
  data?: OrganizationsOrganization | null;
}>;

export function useIdentifyOrganizationGroup({
  accountId,
  organizationKey,
  impersonating = false,
  refreshOrganization,
}: {
  accountId?: string;
  organizationKey?: string;
  impersonating?: boolean;
  refreshOrganization?: RefreshOrganization;
}): void {
  useEffect(() => {
    if (impersonating || !accountId || !organizationKey || !refreshOrganization) {
      return;
    }

    let cancelled = false;
    void refreshOrganization()
      .then((result) => {
        if (cancelled) {
          return;
        }
        const metadata = result.data?.metadata;
        if (!metadata?.id || !metadata.name) {
          return;
        }
        identifyOrganizationGroup({
          id: metadata.id,
          name: metadata.name,
          slug: metadata.slug,
        });
      })
      .catch(() => undefined);

    return () => {
      cancelled = true;
    };
  }, [accountId, organizationKey, impersonating, refreshOrganization]);
}
