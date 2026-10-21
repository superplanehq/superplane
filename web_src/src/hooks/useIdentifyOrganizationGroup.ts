import { useEffect } from "react";

import { identifyOrganizationGroup } from "@/posthog";

export function useIdentifyOrganizationGroup(
  accountId: string | undefined,
  organization: { id?: string; name?: string; slug?: string } | undefined,
): void {
  const id = organization?.id;
  const name = organization?.name;
  const slug = organization?.slug;

  useEffect(() => {
    if (!accountId || !id || !name) {
      return;
    }
    identifyOrganizationGroup({ id, name, slug });
  }, [accountId, id, name, slug]);
}
