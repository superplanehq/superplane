import { useEffect } from "react";
import { useLocation } from "react-router";
import { useOrganization } from "@/hooks/useOrganizationData";

export function OrganizationSuspensionBanner({ organizationId }: { organizationId: string }) {
  const { data: organization, refetch } = useOrganization(organizationId);
  const location = useLocation();
  useEffect(() => {
    void refetch();
  }, [location.pathname, refetch]);
  useEffect(() => {
    const refresh = () => {
      void refetch();
    };
    window.addEventListener("organization-suspended", refresh);
    return () => window.removeEventListener("organization-suspended", refresh);
  }, [refetch]);
  if (!organization?.suspended) return null;
  return (
    <div role="status" className="border-b bg-muted px-4 py-3 text-sm">
      This organization is suspended. Existing data remains available. Configuration changes and new runs are blocked.
      Contact an installation admin.
    </div>
  );
}
