import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useSecrets } from "@/hooks/useSecrets";
import { FieldRendererProps } from "./types";
import { toTestId } from "@/utils/testID";
import type { AuthorizationDomainType } from "@/api-client";

interface SecretFieldRendererProps extends FieldRendererProps {
  domainId: string;
  domainType: AuthorizationDomainType;
}

export const SecretFieldRenderer = ({
  field,
  value,
  onChange,
  domainId,
  domainType,
}: SecretFieldRendererProps) => {
  const { data: secrets, isLoading, error } = useSecrets(domainId, domainType);

  if (!domainId || !domainType) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Secret field requires domainId and domainType props
      </div>
    );
  }

  if (error) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load secrets: {error instanceof Error ? error.message : "Unknown error"}
      </div>
    );
  }

  if (isLoading) {
    return <div className="text-sm text-gray-500 dark:text-gray-400">Loading secrets...</div>;
  }

  if (!secrets || secrets.length === 0) {
    return (
      <div className="space-y-2">
        <Select disabled>
          <SelectTrigger className="w-full" data-testid={field.name ? toTestId(`field-${field.name}-secret`) : undefined}>
            <SelectValue placeholder="No secrets available" />
          </SelectTrigger>
        </Select>
        <p className="text-xs text-gray-500 dark:text-gray-400">
          No secrets found. Create a secret in organization settings first.
        </p>
      </div>
    );
  }

  const testId = field.name ? toTestId(`field-${field.name}-secret`) : undefined;

  return (
    <Select value={(value as string) ?? ""} onValueChange={(val) => onChange(val || undefined)}>
      <SelectTrigger className="w-full" data-testid={testId}>
        <SelectValue placeholder={`Select ${field.label || field.name || "secret"}`} />
      </SelectTrigger>
      <SelectContent className="max-h-60">
        <SelectItem value="">None</SelectItem>
        {secrets.map((secret) => {
          const id = secret.metadata?.id ?? "";
          const name = secret.metadata?.name ?? id;
          if (!id) return null;
          return (
            <SelectItem key={id} value={id}>
              {name}
            </SelectItem>
          );
        })}
      </SelectContent>
    </Select>
  );
};
