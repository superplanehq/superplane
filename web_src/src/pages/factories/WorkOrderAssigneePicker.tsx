import { Avatar } from "@/components/Avatar/avatar";
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { usePermissions } from "@/contexts/usePermissions";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import { buildOrgUserDisplayMap, getUserInitials, resolveOrgUserDisplay } from "@/lib/orgUserDisplay";
import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import { useMemo } from "react";

import { useOptionalFactoriesLayout } from "./layout/factoriesLayoutContext";
import { useFactoryOwnerEligibility } from "@/hooks/useFactoryOwnerEligibility";
import type { FactoriesListFactoryOwnerEligibilityResponse } from "@/api-client";

import { useFactoryTaskIdentity } from "./factoryTaskContext";
import { EmptyOwnerMark } from "./OrgUserReference";

interface WorkOrderAssigneePickerProps {
  organizationId: string;
  factoryId?: string;
  orderId?: string;
  eligibility?: FactoriesListFactoryOwnerEligibilityResponse;
  selectedIds: string[];
  onChange: (assigneeIds: string[]) => void;
  disabled?: boolean;
}

export function WorkOrderAssigneePicker(props: WorkOrderAssigneePickerProps) {
  const layout = useOptionalFactoriesLayout();
  const task = useFactoryTaskIdentity();
  const factoryId = props.factoryId ?? task.factoryId ?? layout?.factoryId;
  if (factoryId && layout?.factory?.onboarding?.codexSubscription === true)
    return <ScopedWorkOrderAssigneePicker {...props} orderId={props.orderId ?? task.orderId} factoryId={factoryId} />;
  return <OwnerPickerOptions {...props} />;
}

function ScopedWorkOrderAssigneePicker(props: WorkOrderAssigneePickerProps & { factoryId: string }) {
  const eligibility = useFactoryOwnerEligibility(props.organizationId, props.factoryId, props.orderId);
  return (
    <>
      {eligibility.isError ? (
        <p role="alert" className="p-3 text-sm">
          Could not load eligible owners. Close and reopen the picker.
        </p>
      ) : null}
      <OwnerPickerOptions
        {...props}
        eligibility={eligibility.data}
        disabled={props.disabled || eligibility.isPending || eligibility.isError}
      />
    </>
  );
}

function OwnerPickerOptions({
  organizationId,
  selectedIds,
  eligibility,
  onChange,
  disabled = false,
}: WorkOrderAssigneePickerProps) {
  const { data: users = [], isLoading } = useOrganizationUsers(organizationId);
  const { currentUserId } = usePermissions();

  disabled = disabled || eligibility?.ownershipChangeAllowed === false;

  const userOptions = useMemo(() => {
    const usersById = buildOrgUserDisplayMap(users);

    const listed = users
      .filter((user) => user.metadata?.id)
      .map((user) => {
        const id = user.metadata!.id!;
        const display = resolveOrgUserDisplay(usersById, id) ?? {
          id,
          name: user.metadata?.email || id,
          initials: getUserInitials(user.metadata?.email || id),
        };

        return {
          id,
          label: display.name,
          display,
        };
      })
      .sort((left, right) => {
        const leftIsSignedInUser = left.id === currentUserId;
        const rightIsSignedInUser = right.id === currentUserId;
        if (leftIsSignedInUser !== rightIsSignedInUser) {
          return leftIsSignedInUser ? -1 : 1;
        }

        return left.label.localeCompare(right.label);
      });
    for (const id of selectedIds) {
      if (!listed.some((user) => user.id === id)) {
        listed.push({ id, label: "Unavailable member", display: { id, name: "Unavailable member", initials: "?" } });
      }
    }
    return listed;
  }, [currentUserId, users, selectedIds]);

  const selectUser = (userId: string) => {
    if (disabled) {
      return;
    }
    onChange([userId]);
  };

  const clearOwner = () => {
    if (disabled) {
      return;
    }
    onChange([]);
  };

  return (
    <Command>
      {eligibility?.restrictionReason ? <p className="p-3 text-sm">{eligibility.restrictionReason}</p> : null}
      <CommandInput placeholder="Search people" autoFocus disabled={disabled} />
      <CommandList>
        {isLoading ? (
          <p className="py-6 text-center text-sm text-muted-foreground">Loading members…</p>
        ) : (
          <>
            <CommandEmpty>No people found.</CommandEmpty>
            <CommandGroup>
              <CommandItem value="No owner" onSelect={clearOwner} disabled={disabled}>
                <EmptyOwnerMark />
                <span className="min-w-0 flex-1 truncate">No owner</span>
                {selectedIds.length === 0 ? <Check className="size-4" aria-hidden /> : null}
              </CommandItem>
              {userOptions.map((user) => {
                const selected = selectedIds.includes(user.id);
                const member = eligibility?.members?.find((member) => member.userId === user.id);
                const ineligible = eligibility?.subscription === true && member?.eligible !== true;
                return (
                  <CommandItem
                    key={user.id}
                    value={`${user.label} ${user.id}`}
                    onSelect={() => selectUser(user.id)}
                    disabled={disabled || ineligible}
                    title={ineligible ? member?.reason || "This member is unavailable." : undefined}
                  >
                    <Avatar src={user.display.avatarUrl} initials={user.display.initials} alt="" className="size-6" />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate">{user.label}</span>
                      {ineligible ? (
                        <span className="block text-xs text-muted-foreground">
                          {member?.reason || "This member is unavailable."}
                        </span>
                      ) : null}
                    </span>
                    <Check className={cn("size-4", selected ? "opacity-100" : "opacity-0")} aria-hidden />
                  </CommandItem>
                );
              })}
            </CommandGroup>
          </>
        )}
      </CommandList>
    </Command>
  );
}
