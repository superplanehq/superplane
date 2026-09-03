import { useAccountOrganizations } from "@/hooks/useAccountOrganizations";
import {
  Building2,
  Check,
  Plus,
} from "lucide-react";
import { useNavigate } from "react-router";

import {
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
} from "@/ui/dropdownMenu";

interface OrganizationSwitchMenuProps {
  currentOrganizationSlug: string;
  onNavigate?: () => void;
  testIdPrefix?: string;
}

/** Shared organization choices for the Factories and legacy navigation menus. */
export function OrganizationSwitchMenu({
  currentOrganizationSlug,
  onNavigate,
  testIdPrefix = "organization",
}: OrganizationSwitchMenuProps) {
  const navigate = useNavigate();
  const organizationsQuery = useAccountOrganizations();
  const organizations = organizationsQuery.data ?? [];

  const goToOrganization = (organizationSlug: string) => {
    if (organizationSlug !== currentOrganizationSlug) {
      navigate(`/${organizationSlug}`);
    }
    onNavigate?.();
  };

  return (
    <>
      <DropdownMenuLabel>Switch organization</DropdownMenuLabel>
      <div>
        {organizationsQuery.isLoading ? <p className="px-2 py-1 text-sm text-muted-foreground">Loading organizations...</p> : null}
        {organizationsQuery.isError ? <p className="px-2 py-1 text-sm text-muted-foreground">Could not load organizations.</p> : null}
        {!organizationsQuery.isLoading && !organizationsQuery.isError && organizations.length === 0 ? (
          <p className="px-2 py-1 text-sm text-muted-foreground">No organizations available.</p>
        ) : null}
        {organizations.map((organization) => {
          const isCurrent = organization.slug === currentOrganizationSlug;
          return (
            <DropdownMenuItem
              key={organization.id}
              onSelect={() => goToOrganization(organization.slug)}
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
      <DropdownMenuItem onSelect={() => navigate("/create")} data-testid={`${testIdPrefix}-organization-create`}>
        <Plus className="h-3.5 w-3.5" aria-hidden />
        Create new organization
      </DropdownMenuItem>
    </>
  );
}
