import { MultiCombobox, MultiComboboxLabel } from "@/components/MultiCombobox/multi-combobox";
import { Button } from "@/components/ui/button";
import { useIntegrationResources } from "@/hooks/useIntegrations";

import { LINEAR_INTAKE_SETUP_COPY } from "./linearIntakeSetupCopy";

export function LinearLabelField({
  labels,
  onChange,
  organizationId = "",
  integrationId = "",
  teamId = "",
}: {
  labels: string[];
  onChange: (labels: string[]) => void;
  organizationId?: string;
  integrationId?: string;
  /** Shows the labels of this team. An intake without a team shows the workspace labels. */
  teamId?: string;
}) {
  const labelScope: Record<string, string> = teamId ? { team: teamId } : { scope: "workspace" };
  const labelsQuery = useIntegrationResources(organizationId, integrationId, "label", labelScope);
  const choices = [
    ...new Set([...(labelsQuery.data ?? []).map((resource) => resource.name ?? "").filter(Boolean), ...labels]),
  ].map((id) => ({ id }));

  return (
    <div className="flex flex-col gap-2" data-testid="linear-intake-label-options">
      {labelsQuery.isLoading ? (
        <p className="workspace-body-text text-muted-foreground">{LINEAR_INTAKE_SETUP_COPY.labelsLoading}</p>
      ) : null}
      {labelsQuery.isError ? (
        <div className="space-y-2">
          <p className="workspace-body-text text-destructive">{LINEAR_INTAKE_SETUP_COPY.labelsError}</p>
          <Button type="button" variant="outline" size="sm" onClick={() => void labelsQuery.refetch()}>
            {LINEAR_INTAKE_SETUP_COPY.wizardRetry}
          </Button>
        </div>
      ) : null}
      {!labelsQuery.isLoading && !labelsQuery.isError && choices.length === 0 ? (
        <p className="workspace-body-text text-muted-foreground">{LINEAR_INTAKE_SETUP_COPY.labelsEmpty}</p>
      ) : null}
      {!labelsQuery.isLoading && !labelsQuery.isError ? (
        <MultiCombobox
          aria-label={LINEAR_INTAKE_SETUP_COPY.labelSearch}
          placeholder={LINEAR_INTAKE_SETUP_COPY.labelSearch}
          options={choices}
          value={labels.map((id) => ({ id }))}
          displayValue={(option) => option.id}
          filter={(option, query) => option.id.toLowerCase().includes(query.trim().toLowerCase())}
          allowCustomValues
          createCustomValue={(query) => ({ id: query.trim() })}
          validateInput={(query) =>
            query.trim().length > 0 && !choices.some((option) => option.id.toLowerCase() === query.trim().toLowerCase())
          }
          onChange={(options) => onChange(options.map((option) => option.id))}
        >
          {(option) => <MultiComboboxLabel>{option.id}</MultiComboboxLabel>}
        </MultiCombobox>
      ) : null}
    </div>
  );
}
