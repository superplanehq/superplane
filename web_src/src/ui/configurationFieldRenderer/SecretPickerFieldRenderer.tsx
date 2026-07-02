import { useMemo } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useSecrets } from "@/hooks/useSecrets";
import type { AuthorizationDomainType } from "@/api-client";

const DOMAIN_TYPE_ORG: AuthorizationDomainType = "DOMAIN_TYPE_ORGANIZATION";
const CLEAR_OPTION_VALUE = "__none__";

interface SecretPickerFieldRendererProps {
  id?: string;
  placeholder?: string;
  required?: boolean;
  value: string | undefined;
  onChange: (value: string) => void;
  organizationId: string | undefined;
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
}: SecretPickerFieldRendererProps) => {
  const domainId = organizationId ?? "";
  // The install page lives outside the /:orgId route, so withOrganizationHeader
  // cannot infer the org from the URL — pass it explicitly.
  const { data: secrets = [], isLoading, error } = useSecrets(domainId, DOMAIN_TYPE_ORG, organizationId);

  const options = useMemo(
    () =>
      secrets
        .map((secret) => {
          const name = secret.metadata?.name ?? "";
          return name ? { value: name, label: name } : null;
        })
        .filter((option): option is { value: string; label: string } => option !== null),
    [secrets],
  );

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

  if (options.length === 0) {
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

  const allowClear = !required;
  const displayValue = value && value.length > 0 ? value : allowClear ? CLEAR_OPTION_VALUE : "";

  return (
    <Select
      value={displayValue}
      onValueChange={(next) => {
        onChange(next === CLEAR_OPTION_VALUE ? "" : next);
      }}
    >
      <SelectTrigger id={id} className="h-8 text-xs">
        <SelectValue placeholder={placeholder ?? "Select a secret"} />
      </SelectTrigger>
      <SelectContent>
        {allowClear ? <SelectItem value={CLEAR_OPTION_VALUE}>None</SelectItem> : null}
        {options.map((option) => (
          <SelectItem key={option.value} value={option.value}>
            {option.label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};
