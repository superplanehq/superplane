import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useOrganizationUsers } from "../../hooks/useOrganizationData";
import { ConfigurationField } from "../../api-client";

interface UserFieldRendererProps {
  field: ConfigurationField;
  value: string;
  onChange: (value: string | undefined) => void;
  domainId: string;
  allValues?: Record<string, unknown>;
}

export const UserFieldRenderer = ({ value, onChange, domainId, allValues }: UserFieldRendererProps) => {
  // Fetch users from the organization
  const { data: users, isLoading, error } = useOrganizationUsers(domainId);
  const approvalContext = getApprovalListContext(allValues);
  const usedUserIds = new Set<string>();

  if (approvalContext) {
    approvalContext.items.forEach((item, index) => {
      if (!item || typeof item !== "object" || index === approvalContext.itemIndex) return;
      const record = item as Record<string, unknown>;
      if (record.type === "user" && typeof record.user === "string" && record.user.trim()) {
        usedUserIds.add(record.user);
      }
    });
  }

  if (!domainId || domainId.trim() === "") {
    return <div className="text-sm text-red-500 dark:text-red-400">User field requires domainId prop</div>;
  }

  if (error) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load users: {error instanceof Error ? error.message : "Unknown error"}
      </div>
    );
  }

  if (isLoading) {
    return <div className="text-sm text-gray-500 dark:text-gray-400">Loading users...</div>;
  }

  if (!users || users.length === 0) {
    return (
      <div className="space-y-2">
        <Select disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="No users available" />
          </SelectTrigger>
        </Select>
        <p className="text-xs text-gray-500 dark:text-gray-400">No users found in this organization.</p>
      </div>
    );
  }

  return (
    <Select value={value ?? ""} onValueChange={(val) => onChange(val || undefined)}>
      <SelectTrigger className="w-full">
        <SelectValue placeholder="Select user" />
      </SelectTrigger>
      <SelectContent>
        {users
          .filter((user) => user.metadata?.id && user.metadata.id.trim() !== "")
          .map((user) => {
            const userId = user.metadata?.id || "";
            const isAlreadyRequested = approvalContext ? usedUserIds.has(userId) && userId !== value : false;
            const label = user.metadata?.email || user.spec?.displayName || user.metadata!.id;
            return (
              <SelectItem key={user.metadata!.id} value={user.metadata!.id!} disabled={isAlreadyRequested}>
                {label}
                {isAlreadyRequested ? " (already requested)" : ""}
              </SelectItem>
            );
          })}
      </SelectContent>
    </Select>
  );
};

function getApprovalListContext(allValues?: Record<string, unknown>): {
  items: Array<Record<string, unknown>>;
  itemIndex: number;
} | null {
  if (!allValues || allValues.__isApprovalList !== true) return null;
  const items = Array.isArray(allValues.__listItems) ? (allValues.__listItems as Array<Record<string, unknown>>) : null;
  const itemIndex = typeof allValues.__itemIndex === "number" ? allValues.__itemIndex : -1;
  if (!items) return null;
  return { items, itemIndex };
}
