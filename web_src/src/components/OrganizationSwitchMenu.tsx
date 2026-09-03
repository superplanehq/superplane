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
  currentOrganizationId: string;
  onNavigate?: () => void;
  testIdPrefix?: string;
}

/** Shared organization choices for the Factories and legacy navigation menus. */
export function OrganizationSwitchMenu({
  currentOrganizationId,
  onNavigate,
  testIdPrefix = "organization",
}: OrganizationSwitchMenuProps) {
  const navigate = useNavigate();
  const { data: organizations = [] } = useAccountOrganizations();

  const goToOrganization = (organizationId: string, organizationSlug: string) => {
    if (organizationId !== currentOrganizationId) {
      navigate(`/${organizationSlug}`);
    }
    onNavigate?.();
  };

  return (
    <>
      <DropdownMenuLabel>Switch organization</DropdownMenuLabel>
      <div>
        {organizations.map((organization) => {
          const isCurrent = organization.id === currentOrganizationId;
          return (
            <DropdownMenuItem
              key={organization.id}
              onSelect={() => goToOrganization(organization.id, organization.slug)}
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
