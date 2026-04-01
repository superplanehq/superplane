import React from "react";
import { useQueries } from "@tanstack/react-query";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useSecrets } from "@/hooks/useSecrets";
import { secretKeys } from "@/hooks/useSecrets";
import { secretsDescribeSecret } from "@/api-client/sdk.gen";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import type { AuthorizationDomainType, ConfigurationField } from "@/api-client";

const SECRET_KEY_DELIMITER = "::";
const LABEL_SEPARATOR = " / ";
const CLEAR_OPTION_VALUE = "__none__";
const DOMAIN_TYPE_ORG: AuthorizationDomainType = "DOMAIN_TYPE_ORGANIZATION";

export type SecretKeyRefValue = { secret: string; key: string } | undefined;

interface SecretKeyFieldRendererProps {
  field: ConfigurationField;
  isRequired: boolean;
  value: SecretKeyRefValue;
  onChange: (value: { secret: string; key: string } | undefined) => void;
  organizationId: string | undefined;
}

/** Value to option value "secret::key" for the dropdown. */
function toOptionValue(value: SecretKeyRefValue): string {
  if (value?.secret && value?.key) return `${value.secret}${SECRET_KEY_DELIMITER}${value.key}`;
  return "";
}

function parseSecretKeySelection(value: string): SecretKeyRefValue {
  if (!value || value === CLEAR_OPTION_VALUE) {
    return undefined;
  }

  const idx = value.indexOf(SECRET_KEY_DELIMITER);
  if (idx < 0) {
    return undefined;
  }

  return { secret: value.slice(0, idx), key: value.slice(idx + SECRET_KEY_DELIMITER.length) };
}

function getDisplayValue(optionValue: string, allowClear: boolean): string {
  if (optionValue) {
    return optionValue;
  }

  return allowClear ? CLEAR_OPTION_VALUE : "";
}

function getSecretKeyPlaceholder(field: ConfigurationField, isRequired: boolean): string {
  if (!isRequired) {
    return "None";
  }

  return field.placeholder ?? "Select credential";
}

function SecretKeyOptions({
  allowClear,
  options,
}: {
  allowClear: boolean;
  options: Array<{ value: string; label: string }>;
}) {
  return (
    <>
      {allowClear ? <SelectItem value={CLEAR_OPTION_VALUE}>None</SelectItem> : null}
      {options.map((opt) => (
        <SelectItem key={opt.value} value={opt.value}>
          {opt.label}
        </SelectItem>
      ))}
    </>
  );
}

/**
 * Single dropdown listing all secret/key variants as "secret-name / key-name".
 * Value is stored as { secret: string, key: string } (YAML-friendly).
 */
export const SecretKeyFieldRenderer = ({
  field,
  isRequired,
  value,
  onChange,
  organizationId,
}: SecretKeyFieldRendererProps) => {
  const domainId = organizationId ?? "";
  const allowClear = !isRequired;
  const placeholder = getSecretKeyPlaceholder(field, isRequired);
  const optionValue = toOptionValue(value);
  const displayValue = getDisplayValue(optionValue, allowClear);
  const { data: secrets = [], isLoading: secretsLoading, error: secretsError } = useSecrets(domainId, DOMAIN_TYPE_ORG);

  const secretRefs = React.useMemo(
    () => secrets.map((s) => s.metadata?.name ?? s.metadata?.id ?? "").filter((ref) => ref.length > 0),
    [secrets],
  );

  const detailQueries = useQueries({
    queries: secretRefs.map((secretRef) => ({
      queryKey: secretKeys.detail(domainId, DOMAIN_TYPE_ORG, secretRef),
      queryFn: async () => {
        const response = await secretsDescribeSecret(
          withOrganizationHeader({
            query: { domainType: DOMAIN_TYPE_ORG, domainId },
            path: { idOrName: secretRef },
          }),
        );
        return response.data?.secret ?? null;
      },
      staleTime: 5 * 60 * 1000,
      gcTime: 10 * 60 * 1000,
      enabled: !!domainId && !!secretRef,
    })),
  });

  const detailByRef = React.useMemo(() => {
    const map: Record<string, (typeof detailQueries)[0]["data"]> = {};
    secretRefs.forEach((ref, index) => {
      map[ref] = detailQueries[index]?.data ?? null;
    });
    return map;
  }, [secretRefs, detailQueries]);

  const options = React.useMemo(() => {
    const list: { value: string; label: string }[] = [];
    secrets.forEach((secret) => {
      const ref = secret.metadata?.name ?? secret.metadata?.id ?? "";
      const secretName = secret.metadata?.name ?? secret.metadata?.id ?? "Unnamed";
      if (!ref) return;
      const detail = detailByRef[ref];
      const keyNames = detail?.spec?.local?.data ? Object.keys(detail.spec.local.data) : [];
      if (keyNames.length === 0) return;
      keyNames.forEach((keyName) => {
        list.push({
          value: `${ref}${SECRET_KEY_DELIMITER}${keyName}`,
          label: `${secretName}${LABEL_SEPARATOR}${keyName}`,
        });
      });
    });
    return list;
  }, [secrets, detailByRef]);

  const detailsLoading = detailQueries.some((q) => q.isLoading);
  const isLoading = secretsLoading || (secrets.length > 0 && detailsLoading);

  if (!organizationId || organizationId.trim() === "") {
    return <div className="text-sm text-red-500 dark:text-red-400">This field requires an organization context.</div>;
  }

  if (secretsError) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load credentials: {secretsError instanceof Error ? secretsError.message : "Unknown error"}
      </div>
    );
  }

  if (isLoading) {
    return (
      <Select disabled>
        <SelectTrigger className="w-full">
          <SelectValue placeholder="Loading…" />
        </SelectTrigger>
      </Select>
    );
  }

  if (!secrets.length || options.length === 0) {
    return (
      <div className="space-y-2">
        <Select disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="No credentials available" />
          </SelectTrigger>
        </Select>
        <p className="text-xs text-muted-foreground">Create credentials in Organization settings first.</p>
      </div>
    );
  }

  return (
    <Select
      value={displayValue}
      onValueChange={(val) => {
        onChange(parseSecretKeySelection(val));
      }}
    >
      <SelectTrigger className="w-full">
        <SelectValue placeholder={placeholder} />
      </SelectTrigger>
      <SelectContent>
        <SecretKeyOptions allowClear={allowClear} options={options} />
      </SelectContent>
    </Select>
  );
};
