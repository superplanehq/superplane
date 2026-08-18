import { useMemo, useState } from "react";
import { Plus } from "lucide-react";
import { Select, SelectContent, SelectItem, SelectSeparator, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { ConfigurationField, SuperplaneSecretsSecret } from "@/api-client";
import { useSecrets } from "@/hooks/useSecrets";
import { usePermissions } from "@/contexts/usePermissions";
import { toTestId } from "@/lib/testID";
import { CreateSecretDialog, type CreatedSecretSummary } from "@/ui/CreateSecretDialog";

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
const ADD_NEW_OPTION_VALUE = "__add_new__";
const DOMAIN_TYPE_ORG = "DOMAIN_TYPE_ORGANIZATION" as const;

function getSecretName(secret: SuperplaneSecretsSecret): string {
  return secret.metadata?.name ?? secret.metadata?.id ?? "";
}

function displaySelectValue(value: string, allowClear: boolean) {
  if (value.length > 0) return value;
  return allowClear ? CLEAR_OPTION_VALUE : "";
}

function SecretFieldEmptyState({ field }: { field: ConfigurationField }) {
  return (
    <div data-testid={toTestId(`secret-field-${field.name}`)} className="space-y-2">
      <Select value="" disabled>
        <SelectTrigger className="w-full">
          <SelectValue placeholder="No secrets available" />
        </SelectTrigger>
      </Select>
      <p className="text-xs text-gray-500 dark:text-gray-400">Create a secret in Organization settings first.</p>
    </div>
  );
}

function SecretFieldOptions({
  isRequired,
  options,
  canCreate,
}: {
  isRequired: boolean;
  options: string[];
  canCreate: boolean;
}) {
  return (
    <>
      {!isRequired ? <SelectItem value={CLEAR_OPTION_VALUE}>None</SelectItem> : null}
      {options.map((option) => (
        <SelectItem key={option} value={option}>
          {option}
        </SelectItem>
      ))}
      {canCreate ? (
        <>
          {options.length > 0 ? <SelectSeparator /> : null}
          <SelectItem value={ADD_NEW_OPTION_VALUE} data-testid="secret-field-add-new-option">
            <span className="flex items-center gap-1.5 text-blue-600 dark:text-blue-400">
              <Plus className="size-3" />
              Add a new secret
            </span>
          </SelectItem>
        </>
      ) : null}
    </>
  );
}

export function SecretFieldRenderer({
  field,
  isRequired,
  value,
  onChange,
  organizationId,
  readOnly = false,
}: SecretFieldRendererProps) {
  const { canAct } = usePermissions();
  const canCreateSecrets = canAct("secrets", "create");
  const [isCreateOpen, setIsCreateOpen] = useState(false);
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

  const handleSecretCreated = (created: CreatedSecretSummary) => {
    if (created.name) {
      onChange({ secret: created.name });
    }
  };

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

  if (options.length === 0 && !canCreateSecrets) {
    return <SecretFieldEmptyState field={field} />;
  }

  const placeholder = isRequired ? (field.placeholder ?? "Select secret") : "None";

  return (
    <div data-testid={toTestId(`secret-field-${field.name}`)}>
      <Select
        value={displaySelectValue(selectedValue, !isRequired)}
        onValueChange={(nextValue) => {
          if (nextValue === ADD_NEW_OPTION_VALUE) {
            setIsCreateOpen(true);
            return;
          }

          if (nextValue === CLEAR_OPTION_VALUE) {
            onChange(undefined);
            return;
          }

          onChange({ secret: nextValue });
        }}
        disabled={readOnly}
      >
        <SelectTrigger className="w-full">
          {/* Render the stored name directly so a just-created secret shows
              before the secrets query refetches (mirrors IntegrationFieldRenderer). */}
          <SelectValue placeholder={placeholder}>{selectedValue || undefined}</SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SecretFieldOptions isRequired={isRequired} options={options} canCreate={canCreateSecrets} />
        </SelectContent>
      </Select>
      {canCreateSecrets ? (
        <CreateSecretDialog
          open={isCreateOpen}
          onOpenChange={setIsCreateOpen}
          organizationId={organizationId}
          onCreated={handleSecretCreated}
        />
      ) : null}
    </div>
  );
}
