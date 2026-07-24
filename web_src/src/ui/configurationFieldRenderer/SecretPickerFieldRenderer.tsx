import { useMemo, useState } from "react";
import { Plus } from "lucide-react";
import { Select, SelectContent, SelectItem, SelectSeparator, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useSecrets } from "@/hooks/useSecrets";
import { usePermissions } from "@/contexts/usePermissions";
import { CreateSecretDialog, type CreatedSecretSummary } from "@/ui/CreateSecretDialog";
import type { AuthorizationDomainType, SuperplaneSecretsSecret } from "@/api-client";

const DOMAIN_TYPE_ORG: AuthorizationDomainType = "DOMAIN_TYPE_ORGANIZATION";
const CLEAR_OPTION_VALUE = "__none__";
const ADD_NEW_OPTION_VALUE = "__add_new__";

interface SecretPickerFieldRendererProps {
  id?: string;
  placeholder?: string;
  required?: boolean;
  value: string | undefined;
  onChange: (value: string) => void;
  organizationId?: string;
  /** Prefills the key name in the create-secret dialog. */
  suggestedKeyName?: string;
}

function secretNameOptions(secrets: SuperplaneSecretsSecret[]) {
  return secrets
    .map((secret) => {
      const name = secret.metadata?.name ?? "";
      return name ? { value: name, label: name } : null;
    })
    .filter((option): option is { value: string; label: string } => option !== null)
    .sort((a, b) => a.label.localeCompare(b.label));
}

function displaySelectValue(value: string | undefined, allowClear: boolean) {
  if (value && value.length > 0) return value;
  return allowClear ? CLEAR_OPTION_VALUE : "";
}

function SecretPickerEmptyState({ id, canCreate }: { id?: string; canCreate: boolean }) {
  if (canCreate) return null;
  return (
    <div className="space-y-1">
      <Select value="" disabled>
        <SelectTrigger id={id} className="h-8 text-xs">
          <SelectValue placeholder="No secrets available" />
        </SelectTrigger>
      </Select>
      <p className="text-[10px] text-slate-400">Create a secret in Organization settings first.</p>
    </div>
  );
}

function SecretPickerOptions({
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
      {options.map((option) => (
        <SelectItem key={option.value} value={option.value}>
          {option.label}
        </SelectItem>
      ))}
      {canCreate ? (
        <>
          {options.length > 0 ? <SelectSeparator /> : null}
          <SelectItem value={ADD_NEW_OPTION_VALUE} data-testid="secret-picker-add-new-option">
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

/**
 * Dropdown that lists the organization's secrets by name. Used by the app
 * install wizard for `secret_picker` params: the selected value is the secret
 * NAME, which the template author embeds in canvas.yaml via
 * `{{ install_params.x }}` so node secrets resolve at runtime.
 */
export const SecretPickerFieldRenderer = ({
  id,
  placeholder,
  required,
  value,
  onChange,
  organizationId,
  suggestedKeyName,
}: SecretPickerFieldRendererProps) => {
  const domainId = organizationId ?? "";
  const { canAct } = usePermissions();
  const canCreateSecrets = canAct("secrets", "create");
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  // The install page lives outside the /:orgId route, so withOrganizationHeader
  // cannot infer the org from the URL — pass it explicitly.
  const { data: secrets = [], isLoading, error } = useSecrets(domainId, DOMAIN_TYPE_ORG, organizationId);
  const options = useMemo(() => secretNameOptions(secrets), [secrets]);
  const allowClear = !required;

  const handleSecretCreated = (created: CreatedSecretSummary) => {
    if (created.name) onChange(created.name);
  };

  const handleValueChange = (next: string) => {
    if (next === ADD_NEW_OPTION_VALUE) {
      setIsCreateOpen(true);
      return;
    }
    onChange(next === CLEAR_OPTION_VALUE ? "" : next);
  };

  if (!organizationId) {
    return <div className="text-xs text-red-500 dark:text-red-400">Select an organization first.</div>;
  }

  if (error) {
    return (
      <div className="text-xs text-red-500 dark:text-red-400">
        Failed to load secrets: {error instanceof Error ? error.message : "Unknown error"}
      </div>
    );
  }

  if (isLoading) {
    return (
      <Select value="" disabled>
        <SelectTrigger id={id} className="h-8 text-xs">
          <SelectValue placeholder="Loading secrets..." />
        </SelectTrigger>
      </Select>
    );
  }

  if (options.length === 0 && !canCreateSecrets) {
    return <SecretPickerEmptyState id={id} canCreate={canCreateSecrets} />;
  }

  return (
    <>
      <Select value={displaySelectValue(value, allowClear)} onValueChange={handleValueChange}>
        <SelectTrigger id={id} className="h-8 text-xs">
          <SelectValue placeholder={placeholder ?? "Select a secret"} />
        </SelectTrigger>
        <SelectContent>
          <SecretPickerOptions allowClear={allowClear} options={options} canCreate={canCreateSecrets} />
        </SelectContent>
      </Select>
      {canCreateSecrets ? (
        <CreateSecretDialog
          open={isCreateOpen}
          onOpenChange={setIsCreateOpen}
          organizationId={organizationId}
          onCreated={handleSecretCreated}
          initialKeyName={suggestedKeyName}
        />
      ) : null}
    </>
  );
};
