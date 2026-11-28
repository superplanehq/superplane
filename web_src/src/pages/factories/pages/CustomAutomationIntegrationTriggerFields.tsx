import type {
  FactoryCustomAutomationIntegrationTrigger,
  IntegrationsCapabilityDefinition,
  OrganizationsIntegration,
} from "@/api-client";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useAvailableIntegrations, useConnectedIntegrations } from "@/hooks/useIntegrations";
import { parseDefaultValues } from "@/lib/components";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";

import { customAutomationListensForMerge } from "../lib/customAutomation";

type IntegrationOption = { connection: OrganizationsIntegration; triggers: IntegrationsCapabilityDefinition[] };

export function CustomAutomationIntegrationTriggerFields({
  idPrefix,
  organizationId,
  factoryId,
  value,
  onChange,
}: {
  idPrefix: string;
  organizationId: string;
  factoryId: string;
  value?: FactoryCustomAutomationIntegrationTrigger;
  onChange: (value: FactoryCustomAutomationIntegrationTrigger) => void;
}) {
  const options = useIntegrationTriggerOptions(organizationId, factoryId);
  if (options.isLoading) {
    return <p className="text-xs text-muted-foreground">Loading integrations...</p>;
  }
  if (options.data.length === 0) {
    return (
      <p className="text-xs text-muted-foreground" data-testid="custom-automation-integration-empty">
        This workspace has no connected integration with events. Connect an integration in the organization settings.
      </p>
    );
  }
  const selected = options.data.find(({ connection }) => connection.metadata?.id === value?.integrationId);
  const trigger = selected?.triggers.find(({ name }) => name === value?.trigger);
  const configuration = value?.configuration ?? {};
  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor={`${idPrefix}-integration`}>Integration</Label>
        <Select value={value?.integrationId ?? ""} onValueChange={(integrationId) => onChange({ integrationId })}>
          <SelectTrigger id={`${idPrefix}-integration`} className="w-full" data-testid="custom-automation-integration">
            <SelectValue placeholder="Select an integration" />
          </SelectTrigger>
          <SelectContent>
            {options.data.map(({ connection }) => (
              <SelectItem key={connection.metadata?.id} value={connection.metadata!.id!}>
                {connection.metadata?.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      {selected ? (
        <EventSelect
          idPrefix={idPrefix}
          triggers={selected.triggers}
          selected={trigger}
          onSelect={(next) =>
            onChange({ ...value, trigger: next.name, configuration: parseDefaultValues(next.configuration ?? []) })
          }
        />
      ) : null}
      {trigger?.configuration?.map((field) => (
        <ConfigurationFieldRenderer
          key={field.name}
          field={field}
          value={configuration[field.name!]}
          allValues={configuration}
          onChange={(fieldValue) =>
            onChange({ ...value, configuration: { ...configuration, [field.name!]: fieldValue } })
          }
          integrationId={value?.integrationId}
          organizationId={organizationId}
          factoryId={factoryId}
        />
      ))}
      {value && customAutomationListensForMerge(value) ? (
        <p className="text-xs text-muted-foreground" data-testid="custom-automation-integration-merge-notice">
          PR Closure does not close a task in this stage when its pull request merges. Use the steps to move the task.
        </p>
      ) : null}
    </div>
  );
}

function EventSelect({
  idPrefix,
  triggers,
  selected,
  onSelect,
}: {
  idPrefix: string;
  triggers: IntegrationsCapabilityDefinition[];
  selected?: IntegrationsCapabilityDefinition;
  onSelect: (trigger: IntegrationsCapabilityDefinition) => void;
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor={`${idPrefix}-event`}>Event</Label>
      <Select
        value={selected?.name ?? ""}
        onValueChange={(name) => {
          const next = triggers.find((item) => item.name === name);
          if (next) onSelect(next);
        }}
      >
        <SelectTrigger id={`${idPrefix}-event`} className="w-full" data-testid="custom-automation-integration-event">
          <SelectValue placeholder="Select an event" />
        </SelectTrigger>
        <SelectContent>
          {triggers.map((item) => (
            <SelectItem key={item.name} value={item.name!}>
              {item.label || item.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {selected?.description ? <p className="text-xs text-muted-foreground">{selected.description}</p> : null}
    </div>
  );
}

/** Lists connected integrations that are ready and that have at least one enabled trigger. */
function useIntegrationTriggerOptions(organizationId: string, factoryId: string) {
  const connected = useConnectedIntegrations(organizationId, { factoryId });
  const definitions = useAvailableIntegrations({ organizationId });
  const data: IntegrationOption[] = [];
  for (const connection of connected.data ?? []) {
    if (connection.status?.state !== "ready" || !connection.metadata?.id) continue;
    const definition = definitions.data?.find(({ name }) => name === connection.metadata?.integrationName);
    const enabled = connection.status.capabilities;
    const triggers = (definition?.capabilities ?? []).filter(
      ({ type, name }) =>
        type === "TYPE_TRIGGER" &&
        (!enabled?.length || enabled.some((state) => state.name === name && state.state === "STATE_ENABLED")),
    );
    if (triggers.length > 0) data.push({ connection, triggers });
  }
  return { isLoading: connected.isLoading || definitions.isLoading, data };
}
