import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

interface Role {
  metadata?: {
    name?: string;
  };
  spec?: {
    displayName?: string;
    description?: string;
  };
}

interface ServiceAccountRoleSelectProps {
  roles: Role[];
  role: string;
  onRoleChange: (value: string) => void;
  isLoading: boolean;
}

export function ServiceAccountRoleSelect({
  roles,
  role,
  onRoleChange,
  isLoading,
}: ServiceAccountRoleSelectProps) {
  if (isLoading) {
    return (
      <div className="flex justify-center items-center h-12">
        <p className="text-gray-500 dark:text-gray-400">Loading roles...</p>
      </div>
    );
  }

  if (roles.length === 0) {
    return (
      <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
        <p className="text-sm text-yellow-800 dark:text-yellow-200 font-medium">No roles available</p>
        <p className="text-yellow-700 dark:text-yellow-300 mt-1 text-sm">
          Create a role first to assign it to this service account.
        </p>
      </div>
    );
  }

  return (
    <>
      <Select value={role} onValueChange={onRoleChange}>
        <SelectTrigger className="w-full" data-testid="sa-create-role">
          <SelectValue placeholder="Select a role" />
        </SelectTrigger>
        <SelectContent>
          {roles.map((r) => (
            <SelectItem key={r.metadata?.name} value={r.metadata?.name || ""}>
              {r.spec?.displayName || r.metadata?.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {role && (
        <p className="mt-1 text-xs text-gray-500">
          {roles.find((r) => r.metadata?.name === role)?.spec?.description || ""}
        </p>
      )}
    </>
  );
}
