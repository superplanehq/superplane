import { useQueries } from "@tanstack/react-query";
import { secretsDescribeSecret } from "@/api-client/sdk.gen";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useSecrets, secretKeys } from "@/hooks/useSecrets";
import { FieldRendererProps } from "./types";
import { toTestId } from "@/utils/testID";
import { withOrganizationHeader } from "@/utils/withOrganizationHeader";
import type { AuthorizationDomainType } from "@/api-client";

interface SecretAndKeyFieldRendererProps extends FieldRendererProps {
  domainId: string;
  domainType: AuthorizationDomainType;
}

export const SecretAndKeyFieldRenderer = ({
  field,
  value,
  onChange,
  domainId,
  domainType,
}: SecretAndKeyFieldRendererProps) => {
  const { data: secrets = [], isLoading: secretsLoading, error: secretsError } = useSecrets(domainId, domainType);

  const secretDetailsQueries = useQueries({
    queries: secrets.map((secret) => {
      const secretId = secret.metadata?.id ?? "";
      return {
        queryKey: secretKeys.detail(domainId, domainType, secretId),
        queryFn: async () => {
          const response = await secretsDescribeSecret(
            withOrganizationHeader({
              query: { domainType, domainId },
              path: { idOrName: secretId },
            }),
          );
          return response.data?.secret ?? null;
        },
        staleTime: 5 * 60 * 1000,
        enabled: !!domainId && !!secretId,
      };
    }),
  });

  const options: { value: string; label: string }[] = [];
  secrets.forEach((secret, i) => {
    const detail = secretDetailsQueries[i]?.data;
    const keyNames = detail?.spec?.local?.data ? Object.keys(detail.spec.local.data) : [];
    const secretName = secret.metadata?.name ?? secret.metadata?.id ?? "";
    const secretId = secret.metadata?.id ?? "";
    if (!secretId) return;
    keyNames.forEach((keyName) => {
      options.push({
        value: `${secretId}:${keyName}`,
        label: `${secretName} / ${keyName}`,
      });
    });
  });

  const detailsLoading = secretDetailsQueries.some((q) => q.isLoading);

  if (!domainId || !domainType) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Secret & key field requires domainId and domainType props
      </div>
    );
  }

  if (secretsError) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load secrets: {secretsError instanceof Error ? secretsError.message : "Unknown error"}
      </div>
    );
  }

  if (secretsLoading || detailsLoading) {
    return <div className="text-sm text-gray-500 dark:text-gray-400">Loading…</div>;
  }

  if (!secrets.length || options.length === 0) {
    return (
      <Select disabled>
        <SelectTrigger
          className="w-full"
          data-testid={field.name ? toTestId(`field-${field.name}-secret-and-key`) : undefined}
        >
          <SelectValue placeholder="No secrets or keys available" />
        </SelectTrigger>
        <SelectContent />
      </Select>
    );
  }

  const testId = field.name ? toTestId(`field-${field.name}-secret-and-key`) : undefined;
  const currentValue = (value as string) ?? "";
  const SELECT_NONE = "__none__";

  return (
    <Select
      value={currentValue || SELECT_NONE}
      onValueChange={(val) => onChange(val === SELECT_NONE ? undefined : val)}
    >
      <SelectTrigger className="w-full" data-testid={testId}>
        <SelectValue placeholder={`Select ${field.label || "secret & key"}`} />
      </SelectTrigger>
      <SelectContent className="max-h-60">
        <SelectItem value={SELECT_NONE}>None</SelectItem>
        {options.map((opt) => (
          <SelectItem key={opt.value} value={opt.value}>
            {opt.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};
