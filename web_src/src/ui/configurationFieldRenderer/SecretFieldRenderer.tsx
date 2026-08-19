import { useMemo } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { ConfigurationField, SuperplaneSecretsSecret } from "@/api-client";
import { useSecrets } from "@/hooks/useSecrets";
import { toTestId } from "@/lib/testID";

export type SecretRefValue = { secret: string } | undefined;

interface SecretFieldRendererProps {
  field: ConfigurationField;
  isRequired: boolean;
  value: SecretRefValue;
  onChange: (value: SecretRefValue) => void;
  organizationId: string;
  readOnly?: boolean;
}

const CLEAR_OPTION_VALUE = "__none__";
const DOMAIN_TYPE_ORG = "DOMAIN_TYPE_ORGANIZATION" as const;

function getSecretName(secret: SuperplaneSecretsSecret): string {
  return secret.metadata?.name ?? secret.metadata?.id ?? "";
}

export function SecretFieldRenderer({
  field,
  isRequired,
  value,
  onChange,
  organizationId,
  readOnly = false,
}: SecretFieldRendererProps) {
  const { data: secrets = [], isLoading, error } = useSecrets(organizationId, DOMAIN_TYPE_ORG, organizationId);

  const options = useMemo(
    () =>
      [...secrets]
        .map((secret) => getSecretName(secret))
        .filter((name) => name.length > 0)
        .sort((left, right) => left.localeCompare(right)),
    [secrets],
  );

  const selectedValue = value?.secret ?? "";

  if (error) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load secrets: {error instanceof Error ? error.message : "Unknown error"}
      </div>
    );
  }

  if (isLoading) {
    return (
      <div data-testid={toTestId(`secret-field-${field.name}`)}>
        <Select value="" disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Loading secrets..." />
          </SelectTrigger>
        </Select>
      </div>
    );
  }

  const placeholder = isRequired ? (field.placeholder ?? "Select secret") : "None";

  return (
    <div data-testid={toTestId(`secret-field-${field.name}`)}>
      <Select
        value={selectedValue || (isRequired ? "" : CLEAR_OPTION_VALUE)}
        onValueChange={(nextValue) => {
          if (nextValue === CLEAR_OPTION_VALUE) {
            onChange(undefined);
            return;
          }

          onChange({ secret: nextValue });
        }}
        disabled={readOnly}
      >
        <SelectTrigger className="w-full">
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent>
          {!isRequired ? <SelectItem value={CLEAR_OPTION_VALUE}>None</SelectItem> : null}
          {options.map((option) => (
            <SelectItem key={option} value={option}>
              {option}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
