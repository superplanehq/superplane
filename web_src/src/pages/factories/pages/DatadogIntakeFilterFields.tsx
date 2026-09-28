import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { cn } from "@/lib/utils";
import { Check, Loader2 } from "lucide-react";
import { useId, type Dispatch, type SetStateAction } from "react";

import { DATADOG_INTAKE_SETTINGS_COPY } from "./datadogIntakeSettingsCopy";
import { DatadogServicePicker } from "./DatadogServicePicker";
import { addIntakeEnvironment, removeIntakeEnvironment, type IntakeSourceSettings } from "./intakeSourceSettingsModel";
import { DATADOG_INTAKE_SETUP_COPY } from "./datadogIntakeSetupCopy";
import type { LineIntakeSourceId } from "./lineIntakeModel";

export function DatadogIntakeFilterFields({
  sourceId,
  settings,
  onSettingsChange,
  organizationId,
  integrationId,
  resourceId,
}: {
  sourceId: LineIntakeSourceId;
  settings: IntakeSourceSettings;
  onSettingsChange: Dispatch<SetStateAction<IntakeSourceSettings>>;
  organizationId?: string;
  integrationId?: string;
  resourceId?: string;
}) {
  const idPrefix = useId();
  const servicesQuery = useIntegrationResources(organizationId ?? "", integrationId ?? "", "service", undefined, {
    enabled: sourceId === "datadog" && Boolean(organizationId && integrationId),
  });
  const serviceName = settings.datadogService || resourceId || "";
  const environmentsQuery = useIntegrationResources(
    organizationId ?? "",
    integrationId ?? "",
    "environment",
    serviceName ? { service: serviceName } : undefined,
    { enabled: sourceId === "datadog" && Boolean(organizationId && integrationId && serviceName) },
  );
  if (sourceId !== "datadog") {
    return null;
  }

  function update<K extends keyof IntakeSourceSettings>(key: K, value: IntakeSourceSettings[K]) {
    onSettingsChange((current) => ({ ...current, [key]: value }));
  }

  return (
    <div className="flex flex-col gap-6">
      <fieldset className="min-w-0">
        <legend className="workspace-section-title">{DATADOG_INTAKE_SETTINGS_COPY.serviceLabel}</legend>
        <div className="mt-2">
          <DatadogServicePicker
            services={servicesQuery.data ?? []}
            serviceName={serviceName}
            loading={servicesQuery.isLoading}
            error={servicesQuery.isError}
            onChange={(name) => update("datadogService", name)}
            onRetry={() => void servicesQuery.refetch()}
            showNameField={false}
          />
        </div>
      </fieldset>
      <fieldset className="min-w-0">
        <legend className="workspace-section-title">{DATADOG_INTAKE_SETTINGS_COPY.intakeSection}</legend>
        <div className="mt-2 flex flex-col gap-2">
          <IntakeSettingsCheckbox
            id={`${idPrefix}-triggered`}
            title={DATADOG_INTAKE_SETTINGS_COPY.triggered}
            checked={settings.datadogTriggeredAlerts}
            onChange={() => update("datadogTriggeredAlerts", !settings.datadogTriggeredAlerts)}
          />
        </div>
      </fieldset>
      <fieldset className="min-w-0">
        <legend className="workspace-section-title">{DATADOG_INTAKE_SETTINGS_COPY.environmentsLabel}</legend>
        <p className="mt-1 text-[12px] text-muted-foreground">{DATADOG_INTAKE_SETTINGS_COPY.environmentsHelper}</p>
        <div className="mt-2">
          <EnvironmentPicker
            environments={environmentOptions(environmentsQuery.data ?? [], settings.datadogEnvironments)}
            selected={settings.datadogEnvironments}
            loading={environmentsQuery.isLoading}
            error={environmentsQuery.isError}
            onToggle={(name) =>
              update("datadogEnvironments", toggleIntakeEnvironment(settings.datadogEnvironments, name))
            }
            onRetry={() => void environmentsQuery.refetch()}
          />
        </div>
      </fieldset>
    </div>
  );
}

function EnvironmentPicker({
  environments,
  selected,
  loading,
  error,
  onToggle,
  onRetry,
}: {
  environments: string[];
  selected: string[];
  loading: boolean;
  error: boolean;
  onToggle: (name: string) => void;
  onRetry: () => void;
}) {
  return (
    <div className="space-y-3" data-testid="datadog-environment-options">
      {loading ? (
        <p className="workspace-body-text flex items-center gap-2 text-muted-foreground">
          <Loader2 className="size-4 animate-spin" aria-hidden />
          {DATADOG_INTAKE_SETTINGS_COPY.environmentsLoading}
        </p>
      ) : null}
      {error ? (
        <div className="space-y-3">
          <p className="workspace-body-text text-destructive">{DATADOG_INTAKE_SETTINGS_COPY.environmentsError}</p>
          <Button type="button" variant="outline" size="sm" onClick={onRetry}>
            {DATADOG_INTAKE_SETUP_COPY.wizardRetry}
          </Button>
        </div>
      ) : null}
      {!loading && !error && environments.length === 0 ? (
        <p className="workspace-body-text text-muted-foreground">{DATADOG_INTAKE_SETTINGS_COPY.environmentsEmpty}</p>
      ) : null}
      {!loading && !error && environments.length > 0 ? (
        <div
          className="max-h-56 overflow-y-auto rounded-lg border border-border"
          role="listbox"
          aria-label="Environments"
          aria-multiselectable="true"
        >
          <ul className="divide-y divide-border">
            {environments.map((environment) => {
              const checked = selected.includes(environment);
              return (
                <li key={environment}>
                  <Button
                    type="button"
                    variant="ghost"
                    role="option"
                    aria-selected={checked}
                    onClick={() => onToggle(environment)}
                    data-testid={`datadog-environment-${environment}`}
                    className={cn(
                      "h-auto w-full justify-start gap-3 rounded-none px-3 py-2.5 text-left text-[13px] font-medium",
                      checked ? "bg-accent/50" : "hover:bg-accent/30",
                    )}
                  >
                    <span className="min-w-0 flex-1 truncate">{environment}</span>
                    {checked ? (
                      <Check className="size-3.5 shrink-0 text-foreground" strokeWidth={2.5} aria-hidden />
                    ) : null}
                  </Button>
                </li>
              );
            })}
          </ul>
        </div>
      ) : null}
    </div>
  );
}

function environmentOptions(resources: Array<{ id?: string; name?: string }>, selected: string[]): string[] {
  const names = new Set<string>();
  for (const environment of selected) {
    const name = environment.trim().toLowerCase();
    if (name) {
      names.add(name);
    }
  }
  for (const resource of resources) {
    const name = (resource.id ?? resource.name ?? "").trim().toLowerCase();
    if (name) {
      names.add(name);
    }
  }
  return [...names].sort();
}

function toggleIntakeEnvironment(environments: string[], environment: string): string[] {
  const next = environment.trim().toLowerCase();
  if (environments.includes(next)) {
    return removeIntakeEnvironment(environments, next);
  }
  return addIntakeEnvironment(environments, next);
}

function IntakeSettingsCheckbox({
  id,
  title,
  description,
  checked,
  onChange,
}: {
  id: string;
  title: string;
  description?: string;
  checked: boolean;
  onChange: () => void;
}) {
  return (
    <div
      className={cn(
        "flex items-start gap-3 rounded-lg border px-3 py-2.5 transition-colors",
        checked ? "border-foreground/20 bg-accent/50" : "border-border bg-card hover:border-foreground/15",
      )}
    >
      <Checkbox id={id} checked={checked} onChange={onChange} className="mt-0.5" />
      <div className="min-w-0">
        <Label htmlFor={id} className="cursor-pointer text-[13px] font-medium tracking-[-0.01em] text-foreground">
          {title}
        </Label>
        {description ? <p className="mt-0.5 text-[12px] text-muted-foreground">{description}</p> : null}
      </div>
    </div>
  );
}
