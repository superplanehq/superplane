import { useMemo } from "react";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { ConfigurationField, OrganizationsIntegration } from "@/api-client";
import { useConnectedIntegrations } from "@/hooks/useIntegrations";
import { toTestId } from "@/lib/testID";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";

export type IntegrationRefValue = { name: string } | undefined;

interface IntegrationFieldRendererProps {
  field: ConfigurationField;
  isRequired: boolean;
  value: IntegrationRefValue;
  onChange: (value: IntegrationRefValue) => void;
  organizationId: string;
  readOnly?: boolean;
}

const CLEAR_OPTION_VALUE = "__none__";

function getIntegrationTypeFilter(field: ConfigurationField): string | undefined {
  return field.typeOptions?.integration?.integration?.trim() || undefined;
}

function matchesIntegrationType(integration: OrganizationsIntegration, integrationType: string | undefined): boolean {
  if (!integrationType) {
    return true;
  }

  return integration.metadata?.integrationName === integrationType;
}

function filterReadyIntegrations(
  integrations: OrganizationsIntegration[],
  integrationType: string | undefined,
): OrganizationsIntegration[] {
  return integrations.filter((integration) => {
    if (integration.status?.state !== "ready") {
      return false;
    }

    if (!integration.metadata?.name?.trim()) {
      return false;
    }

    return matchesIntegrationType(integration, integrationType);
  });
}

function getInstallationName(integration: OrganizationsIntegration): string {
  return integration.metadata?.name?.trim() ?? "";
}

function IntegrationOptionLabel({ integration }: { integration: OrganizationsIntegration }) {
  const name = getInstallationName(integration) || "Unnamed integration";

  return (
    <span className="flex items-center gap-2">
      <IntegrationIcon
        integrationName={integration.metadata?.integrationName}
        className="h-4 w-4 flex-shrink-0 text-gray-500 dark:text-gray-400"
      />
      <span>{name}</span>
    </span>
  );
}

export function IntegrationFieldRenderer({
  field,
  isRequired,
  value,
  onChange,
  organizationId,
  readOnly = false,
}: IntegrationFieldRendererProps) {
  const integrationType = getIntegrationTypeFilter(field);
  const { data: integrations = [], isLoading, error } = useConnectedIntegrations(organizationId);

  const options = useMemo(
    () =>
      filterReadyIntegrations(integrations, integrationType).sort((left, right) =>
        getInstallationName(left).localeCompare(getInstallationName(right)),
      ),
    [integrations, integrationType],
  );

  const selectedName = value?.name ?? "";
  const selectedIntegration = useMemo(
    () => options.find((integration) => getInstallationName(integration) === selectedName),
    [options, selectedName],
  );

  if (error) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load integrations: {error instanceof Error ? error.message : "Unknown error"}
      </div>
    );
  }

  if (isLoading) {
    return (
      <div data-testid={toTestId(`integration-field-${field.name}`)}>
        <Select value="" disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Loading integrations..." />
          </SelectTrigger>
        </Select>
      </div>
    );
  }

  const placeholder = isRequired ? (field.placeholder ?? "Select integration") : "None";

  return (
    <div data-testid={toTestId(`integration-field-${field.name}`)}>
      <Select
        value={selectedName || (isRequired ? "" : CLEAR_OPTION_VALUE)}
        onValueChange={(nextValue) => {
          if (nextValue === CLEAR_OPTION_VALUE) {
            onChange(undefined);
            return;
          }

          const integration = options.find((item) => getInstallationName(item) === nextValue);
          if (!integration) {
            onChange(undefined);
            return;
          }

          onChange({ name: getInstallationName(integration) });
        }}
        disabled={readOnly}
      >
        <SelectTrigger className="w-full">
          <SelectValue placeholder={placeholder}>
            {selectedIntegration ? <IntegrationOptionLabel integration={selectedIntegration} /> : selectedName}
          </SelectValue>
        </SelectTrigger>
        <SelectContent>
          {!isRequired ? <SelectItem value={CLEAR_OPTION_VALUE}>None</SelectItem> : null}
          {options.map((integration) => {
            const name = getInstallationName(integration);
            return (
              <SelectItem key={name} value={name}>
                <IntegrationOptionLabel integration={integration} />
              </SelectItem>
            );
          })}
        </SelectContent>
      </Select>
    </div>
  );
}
