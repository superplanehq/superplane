import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useSecret } from "@/hooks/useSecrets";
import { FieldRendererProps } from "./types";
import { toTestId } from "@/utils/testID";
import type { AuthorizationDomainType } from "@/api-client";

/**
 * Infers the config field name that holds the secret reference from this key field's name.
 * e.g. authenticationSecretKey -> authenticationSecret, authenticationSecretUsernameKey -> authenticationSecret
 */
function getSecretFieldName(keyFieldName: string): string {
  const match = keyFieldName?.match(/^(.+?)Secret(Key|UsernameKey|PasswordKey)$/);
  return match ? `${match[1]}Secret` : "";
}

interface SecretKeyFieldRendererProps extends FieldRendererProps {
  domainId: string;
  domainType: AuthorizationDomainType;
}

export const SecretKeyFieldRenderer = ({
  field,
  value,
  onChange,
  allValues = {},
  domainId,
  domainType,
}: SecretKeyFieldRendererProps) => {
  const secretFieldName = getSecretFieldName(field.name || "");
  const secretRef = (secretFieldName ? allValues[secretFieldName] : undefined) as string | undefined;
  const { data: secret, isLoading, error } = useSecret(domainId, domainType, secretRef ?? "");

  const keyNames = secret?.spec?.local?.data
    ? Object.keys(secret.spec.local.data)
    : [];

  if (!domainId || !domainType) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Secret key field requires domainId and domainType props
      </div>
    );
  }

  if (!secretFieldName) {
    return (
      <div className="text-sm text-gray-500 dark:text-gray-400">
        Secret key field name could not be inferred
      </div>
    );
  }

  if (!secretRef || secretRef === "") {
    return (
      <Select disabled>
        <SelectTrigger className="w-full" data-testid={field.name ? toTestId(`field-${field.name}-secret-key`) : undefined}>
          <SelectValue placeholder="Select a secret first" />
        </SelectTrigger>
        <SelectContent />
      </Select>
    );
  }

  if (error) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load secret keys: {error instanceof Error ? error.message : "Unknown error"}
      </div>
    );
  }

  if (isLoading) {
    return <div className="text-sm text-gray-500 dark:text-gray-400">Loading keys...</div>;
  }

  const testId = field.name ? toTestId(`field-${field.name}-secret-key`) : undefined;

  return (
    <Select
      value={(value as string) ?? ""}
      onValueChange={(val) => onChange(val || undefined)}
    >
      <SelectTrigger className="w-full" data-testid={testId}>
        <SelectValue placeholder={`Select ${field.label || "key"}`} />
      </SelectTrigger>
      <SelectContent className="max-h-60">
        <SelectItem value="">None</SelectItem>
        {keyNames.map((keyName) => (
          <SelectItem key={keyName} value={keyName}>
            {keyName}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};
