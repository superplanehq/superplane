import { useAccountOrganizations } from "@/hooks/useAccountOrganizations";
import type { AccountOrganization } from "@/lib/accountOrganizations";
import { organizationMatchesRoute, organizationRouteId } from "@/lib/accountOrganizations";
import { Building2, Check, Plus } from "lucide-react";
import { useNavigate } from "react-router";

import { DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator } from "@/ui/dropdownMenu";

interface OrganizationSwitchMenuProps {
  currentOrganizationRouteId: string;
  onNavigate?: () => void;
  testIdPrefix?: string;
}

/** Shared organization choices for the Factories and legacy navigation menus. */
export function OrganizationSwitchMenu({
  currentOrganizationRouteId,
  onNavigate,
  testIdPrefix = "organization",
}: OrganizationSwitchMenuProps) {
  const navigate = useNavigate();
  const organizationsQuery = useAccountOrganizations();
  const organizations = organizationsQuery.data ?? [];

  const goToOrganization = (organization: AccountOrganization) => {
    if (!organizationMatchesRoute(organization, currentOrganizationRouteId)) {
      navigate(`/${organizationRouteId(organization)}`);
    }
    onNavigate?.();
  };

  return (
    <>
      <DropdownMenuLabel>Switch organization</DropdownMenuLabel>
      <div>
        {organizationsQuery.isLoading ? (
          <p className="px-2 py-1 text-sm text-muted-foreground">Loading organizations...</p>
        ) : null}
        {organizationsQuery.isError ? (
          <p className="px-2 py-1 text-sm text-muted-foreground">Could not load organizations.</p>
        ) : null}
        {!organizationsQuery.isLoading && !organizationsQuery.isError && organizations.length === 0 ? (
          <p className="px-2 py-1 text-sm text-muted-foreground">No organizations available.</p>
        ) : null}
        {organizations.map((organization) => {
          const isCurrent = organizationMatchesRoute(organization, currentOrganizationRouteId);
          return (
            <DropdownMenuItem
              key={organization.id}
              onSelect={() => goToOrganization(organization)}
              aria-checked={isCurrent}
              data-testid={`${testIdPrefix}-organization-option-${organization.id}`}
            >
              <Building2 className="h-3.5 w-3.5" aria-hidden />
              <span className="truncate">{organization.name}</span>
              {isCurrent ? <Check className="ml-auto h-3.5 w-3.5" aria-hidden /> : null}
            </DropdownMenuItem>
          );
        })}
      </div>
      <DropdownMenuSeparator />
      <DropdownMenuItem onSelect={() => navigate("/onboarding")} data-testid={`${testIdPrefix}-organization-create`}>
        <Plus className="h-3.5 w-3.5" aria-hidden />
        Create new organization
      </DropdownMenuItem>
    </>
  );
}
