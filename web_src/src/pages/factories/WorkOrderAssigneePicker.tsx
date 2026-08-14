import { Checkbox } from "@/components/ui/checkbox";
import { Avatar } from "@/components/Avatar/avatar";
import { useOrganizationUsers } from "@/hooks/useOrganizationData";
import { buildOrgUserDisplayMap, getUserInitials, resolveOrgUserDisplay } from "@/lib/orgUserDisplay";
import { cn } from "@/lib/utils";
import { useMemo } from "react";

interface WorkOrderAssigneePickerProps {
  organizationId: string;
  selectedIds: string[];
  onChange: (assigneeIds: string[]) => void;
  disabled?: boolean;
  variant?: "default" | "popover";
}

export function WorkOrderAssigneePicker({
  organizationId,
  selectedIds,
  onChange,
  disabled = false,
  variant = "default",
}: WorkOrderAssigneePickerProps) {
  const { data: users = [], isLoading } = useOrganizationUsers(organizationId);

  const userOptions = useMemo(() => {
    const usersById = buildOrgUserDisplayMap(users);

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
        const leftSelected = selectedIds.includes(left.id);
        const rightSelected = selectedIds.includes(right.id);
        if (leftSelected !== rightSelected) {
          return leftSelected ? -1 : 1;
        }
        return left.label.localeCompare(right.label);
      });
  }, [users, selectedIds]);

  const toggleUser = (userId: string, checked: boolean) => {
    if (disabled) {
      return;
    }

    onChange(
      checked
        ? selectedIds.includes(userId)
          ? selectedIds
          : [...selectedIds, userId]
        : selectedIds.filter((id) => id !== userId),
    );
  };

  if (isLoading) {
    return <p className="text-sm text-gray-500 dark:text-gray-400">Loading members…</p>;
  }

  if (userOptions.length === 0) {
    return <p className="text-sm text-gray-500 dark:text-gray-400">No organization members found.</p>;
  }

  return (
    <ul
      className={cn(
        "overflow-y-auto",
        variant === "popover"
          ? "max-h-60 space-y-0.5"
          : "max-h-72 space-y-1 rounded-lg border border-gray-200 p-2 dark:border-gray-700/70",
      )}
    >
      {userOptions.map((user) => {
        const checked = selectedIds.includes(user.id);
        return (
          <li key={user.id}>
            <label
              className={cn(
                "flex cursor-pointer items-center gap-3 rounded-md px-2 py-2 hover:bg-gray-50 dark:hover:bg-gray-800/60",
                checked && "bg-gray-50 dark:bg-gray-800/40",
                disabled && "cursor-not-allowed opacity-60",
              )}
            >
              <Checkbox
                checked={checked}
                disabled={disabled}
                onChange={(event) => toggleUser(user.id, event.target.checked)}
              />
              <Avatar
                src={user.display.avatarUrl}
                initials={user.display.initials}
                alt={user.display.name}
                className="size-6"
              />
              <span className="text-sm text-gray-900 dark:text-gray-100">{user.label}</span>
            </label>
          </li>
        );
      })}
    </ul>
  );
}
