import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../select";
import { useOrganizationGroups } from "../../hooks/useOrganizationData";
import { ConfigurationField } from "../../api-client";

interface GroupFieldRendererProps {
  field: ConfigurationField;
  value: string;
  onChange: (value: string | undefined) => void;
  domainId: string;
  allValues?: Record<string, unknown>;
}

export const GroupFieldRenderer = ({ value, onChange, domainId, allValues }: GroupFieldRendererProps) => {
  // Fetch groups from the organization
  const { data: groups, isLoading, error } = useOrganizationGroups(domainId);
  const approvalContext = getApprovalListContext(allValues);
  const usedGroups = new Set<string>();

  if (approvalContext) {
    approvalContext.items.forEach((item, index) => {
      if (!item || typeof item !== "object" || index === approvalContext.itemIndex) return;
      const record = item as Record<string, unknown>;
      if (record.type === "group" && typeof record.group === "string" && record.group.trim()) {
        usedGroups.add(record.group);
      }
    });
  }

  if (!domainId || domainId.trim() === "") {
    return <div className="text-sm text-red-500 dark:text-red-400">Group field requires domainId prop</div>;
  }

  if (error) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load groups: {error instanceof Error ? error.message : "Unknown error"}
      </div>
    );
  }

  if (isLoading) {
    return <div className="text-sm text-gray-500 dark:text-gray-400">Loading groups...</div>;
  }

  if (!groups || groups.length === 0) {
    return (
      <div className="space-y-2">
        <Select disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="No groups available" />
          </SelectTrigger>
        </Select>
        <p className="text-xs text-gray-500 dark:text-gray-400">No groups found in this organization.</p>
      </div>
    );
  }

  return (
    <Select value={value ?? ""} onValueChange={(val) => onChange(val || undefined)}>
      <SelectTrigger className="w-full">
        <SelectValue placeholder="Select group" />
      </SelectTrigger>
      <SelectContent>
        {groups
          .filter((group) => group.metadata?.name && group.metadata.name.trim() !== "")
          .map((group) => {
            const groupName = group.metadata?.name || "";
            const isAlreadyRequested = approvalContext ? usedGroups.has(groupName) && groupName !== value : false;
            const label = group.spec?.displayName || group.metadata!.name;
            return (
              <SelectItem key={group.metadata!.name} value={group.metadata!.name!} disabled={isAlreadyRequested}>
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
