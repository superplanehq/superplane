import { Avatar } from "@/components/Avatar/avatar";
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import { buildOrgUserDisplayMap, getUserInitials, resolveOrgUserDisplay } from "@/lib/orgUserDisplay";
import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import { useMemo } from "react";

import { EmptyOwnerMark } from "./OrgUserReference";

interface WorkOrderAssigneePickerProps {
  organizationId: string;
  selectedIds: string[];
  /** Ids used to decide sort order (pinned to the top). Defaults to `selectedIds`. */
  pinnedIds?: string[];
  onChange: (assigneeIds: string[]) => void;
  disabled?: boolean;
}

export function WorkOrderAssigneePicker({
  organizationId,
  selectedIds,
  pinnedIds,
  onChange,
  disabled = false,
}: WorkOrderAssigneePickerProps) {
  const { data: users = [], isLoading } = useOrganizationUsers(organizationId);
  const pinned = pinnedIds ?? selectedIds;
  const selectedId = selectedIds[0];

  const userOptions = useMemo(() => {
    const usersById = buildOrgUserDisplayMap(users);
    const pinnedSet = new Set(pinned);

    return users
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
        const leftPinned = pinnedSet.has(left.id);
        const rightPinned = pinnedSet.has(right.id);
        if (leftPinned !== rightPinned) {
          return leftPinned ? -1 : 1;
        }

        return left.label.localeCompare(right.label);
      });
  }, [users, pinned]);

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
                const selected = user.id === selectedId;
                return (
                  <CommandItem
                    key={user.id}
                    value={`${user.label} ${user.id}`}
                    onSelect={() => selectUser(user.id)}
                    disabled={disabled}
                  >
                    <Avatar src={user.display.avatarUrl} initials={user.display.initials} alt="" className="size-6" />
                    <span className="min-w-0 flex-1 truncate">{user.label}</span>
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
