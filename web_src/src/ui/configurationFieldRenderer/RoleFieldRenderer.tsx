import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useOrganizationRoles } from "../../hooks/useOrganizationData";
import { ConfigurationField } from "../../api-client";

interface RoleFieldRendererProps {
  field: ConfigurationField;
  value: string;
  onChange: (value: string | undefined) => void;
  domainId: string;
  allValues?: Record<string, unknown>;
}

export const RoleFieldRenderer = ({ value, onChange, domainId, allValues }: RoleFieldRendererProps) => {
  // Fetch roles from the organization
  const { data: roles, isLoading, error } = useOrganizationRoles(domainId);
  const approvalContext = getApprovalListContext(allValues);
  const usedRoles = new Set<string>();

  if (approvalContext) {
    approvalContext.items.forEach((item, index) => {
      if (!item || typeof item !== "object" || index === approvalContext.itemIndex) return;
      const record = item as Record<string, unknown>;
      if (record.type === "role" && typeof record.role === "string" && record.role.trim()) {
        usedRoles.add(record.role);
      }
    });
  }

  if (!domainId || domainId.trim() === "") {
    return <div className="text-sm text-red-500 dark:text-red-400">Role field requires domainId prop</div>;
  }

  if (error) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load roles: {error instanceof Error ? error.message : "Unknown error"}
      </div>
    );
  }

  if (isLoading) {
    return <div className="text-sm text-gray-500 dark:text-gray-400">Loading roles...</div>;
  }

  if (!roles || roles.length === 0) {
    return (
      <div className="space-y-2">
        <Select disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="No roles available" />
          </SelectTrigger>
        </Select>
        <p className="text-xs text-gray-500 dark:text-gray-400">No roles found in this organization.</p>
      </div>
    );
  }

  return (
    <Select value={value ?? ""} onValueChange={(val) => onChange(val || undefined)}>
      <SelectTrigger className="w-full">
        <SelectValue placeholder="Select role" />
      </SelectTrigger>
      <SelectContent>
        {roles
          .filter((role) => role.metadata?.name && role.metadata.name.trim() !== "")
          .map((role) => {
            const roleName = role.metadata?.name || "";
            const isAlreadyRequested = approvalContext ? usedRoles.has(roleName) && roleName !== value : false;
            const label = role.spec?.displayName || role.metadata!.name;
            return (
              <SelectItem key={role.metadata!.name} value={role.metadata!.name!} disabled={isAlreadyRequested}>
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
