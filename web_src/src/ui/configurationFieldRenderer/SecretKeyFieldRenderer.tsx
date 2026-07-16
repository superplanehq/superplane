import React from "react";
import { Plus } from "lucide-react";
import { useQueries } from "@tanstack/react-query";
import { Select, SelectContent, SelectItem, SelectSeparator, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useSecrets } from "@/hooks/useSecrets";
import { secretKeys } from "@/hooks/useSecrets";
import { secretsDescribeSecret } from "@/api-client/sdk.gen";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { usePermissions } from "@/contexts/usePermissions";
import { CreateSecretDialog, type CreatedSecretSummary } from "@/ui/CreateSecretDialog";
import type { AuthorizationDomainType, ConfigurationField, SuperplaneSecretsSecret } from "@/api-client";

const SECRET_KEY_DELIMITER = "::";
const LABEL_SEPARATOR = " / ";
const CLEAR_OPTION_VALUE = "__none__";
const ADD_NEW_OPTION_VALUE = "__add_new__";
const DOMAIN_TYPE_ORG: AuthorizationDomainType = "DOMAIN_TYPE_ORGANIZATION";

export type SecretKeyRefValue = { secret: string; key: string } | undefined;

interface SecretKeyFieldRendererProps {
  field: ConfigurationField;
  isRequired: boolean;
  value: SecretKeyRefValue;
  onChange: (value: { secret: string; key: string } | undefined) => void;
  organizationId: string | undefined;
  readOnly?: boolean;
}

type SecretDetail = SuperplaneSecretsSecret | null | undefined;

/** Value to option value "secret::key" for the dropdown. */
function toOptionValue(value: SecretKeyRefValue): string {
  if (value?.secret && value?.key) return `${value.secret}${SECRET_KEY_DELIMITER}${value.key}`;
  return "";
}

function parseSecretKeySelection(value: string): SecretKeyRefValue {
  if (!value || value === CLEAR_OPTION_VALUE) return undefined;
  const idx = value.indexOf(SECRET_KEY_DELIMITER);
  if (idx < 0) return undefined;
  return { secret: value.slice(0, idx), key: value.slice(idx + SECRET_KEY_DELIMITER.length) };
}

function getSecretKeyPlaceholder(field: ConfigurationField, isRequired: boolean): string {
  return isRequired ? (field.placeholder ?? "Select credential") : "None";
}

function getSecretRef(secret: SuperplaneSecretsSecret): string {
  return secret.metadata?.name ?? secret.metadata?.id ?? "";
}

function buildOptions(secrets: SuperplaneSecretsSecret[], detailByRef: Record<string, SecretDetail>) {
  const list: { value: string; label: string }[] = [];
  const sortedSecrets = [...secrets].sort((a, b) => getSecretRef(a).localeCompare(getSecretRef(b)));
  sortedSecrets.forEach((secret) => {
    const ref = getSecretRef(secret);
    const secretName = ref || "Unnamed";
    if (!ref) return;
    const detail = detailByRef[ref];
    const data = detail?.spec?.local?.data;
    if (!data) return;
    const keyNames = Object.keys(data).sort((a, b) => a.localeCompare(b));
    keyNames.forEach((keyName) => {
      list.push({
        value: `${ref}${SECRET_KEY_DELIMITER}${keyName}`,
        label: `${secretName}${LABEL_SEPARATOR}${keyName}`,
      });
    });
  });
  return list;
}

function AddNewSecretOption({ showSeparator }: { showSeparator: boolean }) {
  return (
    <>
      {showSeparator && <SelectSeparator />}
      <SelectItem value={ADD_NEW_OPTION_VALUE} data-testid="secret-key-add-new-option">
        <span className="flex items-center gap-1.5 text-blue-600 dark:text-blue-400">
          <Plus className="size-3" />
          Add a new secret
        </span>
      </SelectItem>
    </>
  );
}

function SecretKeyOptions({
  allowClear,
  options,
  canCreate,
}: {
  allowClear: boolean;
  options: Array<{ value: string; label: string }>;
  canCreate: boolean;
}) {
  return (
    <>
      {allowClear ? <SelectItem value={CLEAR_OPTION_VALUE}>None</SelectItem> : null}
      {options.map((opt) => (
        <SelectItem key={opt.value} value={opt.value}>
          {opt.label}
        </SelectItem>
      ))}
      {canCreate && <AddNewSecretOption showSeparator={options.length > 0} />}
    </>
  );
}

function useSecretDetails(secrets: SuperplaneSecretsSecret[], domainId: string) {
  const secretRefs = React.useMemo(() => secrets.map(getSecretRef).filter((ref) => ref.length > 0), [secrets]);

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
    const map: Record<string, SecretDetail> = {};
    secretRefs.forEach((ref, index) => {
      map[ref] = detailQueries[index]?.data ?? null;
    });
    return map;
  }, [secretRefs, detailQueries]);

  const detailsLoading = detailQueries.some((q) => q.isLoading);
  return { detailByRef, detailsLoading };
}

function DisabledEmptyPicker() {
  return (
    <div className="space-y-2">
      <Select value="" disabled>
        <SelectTrigger className="w-full">
          <SelectValue placeholder="No credentials available" />
        </SelectTrigger>
      </Select>
      <p className="text-xs text-muted-foreground">Create credentials in Organization settings first.</p>
    </div>
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
  readOnly = false,
}: SecretKeyFieldRendererProps) => {
  const domainId = organizationId ?? "";
  const allowClear = !isRequired;
  const placeholder = getSecretKeyPlaceholder(field, isRequired);
  const optionValue = toOptionValue(value);
  const displayValue = optionValue || (allowClear ? CLEAR_OPTION_VALUE : "");
  const { canAct } = usePermissions();
  const canCreateSecrets = canAct("secrets", "create");
  const [isCreateOpen, setIsCreateOpen] = React.useState(false);
  const { data: secrets = [], isLoading: secretsLoading, error: secretsError } = useSecrets(domainId, DOMAIN_TYPE_ORG);
  const { detailByRef, detailsLoading } = useSecretDetails(secrets, domainId);

  const options = React.useMemo(() => buildOptions(secrets, detailByRef), [secrets, detailByRef]);
  const isLoading = secretsLoading || (secrets.length > 0 && detailsLoading);

  const handleSecretCreated = (created: CreatedSecretSummary) => {
    if (created.name && created.keys.length > 0) {
      onChange({ secret: created.name, key: created.keys[0] });
    }
  };

  const handleValueChange = (val: string) => {
    if (readOnly) {
      return;
    }

    if (val === ADD_NEW_OPTION_VALUE) {
      setIsCreateOpen(true);
      return;
    }
    onChange(parseSecretKeySelection(val));
  };

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
      <Select value="" disabled>
        <SelectTrigger className="w-full">
          <SelectValue placeholder="Loading…" />
        </SelectTrigger>
      </Select>
    );
  }

  if (options.length === 0 && !canCreateSecrets) {
    return <DisabledEmptyPicker />;
  }

  return (
    <>
      <Select value={displayValue} onValueChange={handleValueChange} disabled={readOnly}>
        <SelectTrigger className="w-full ph-no-capture">
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent>
          <SecretKeyOptions allowClear={allowClear} options={options} canCreate={canCreateSecrets} />
        </SelectContent>
      </Select>
      {canCreateSecrets && (
        <CreateSecretDialog
          open={isCreateOpen}
          onOpenChange={setIsCreateOpen}
          organizationId={organizationId}
          onCreated={handleSecretCreated}
        />
      )}
    </>
  );
};
