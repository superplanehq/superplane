import { useMemo, useState } from "react";
import { Plus } from "lucide-react";
import { Select, SelectContent, SelectItem, SelectSeparator, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { ConfigurationField, OrganizationsIntegration } from "@/api-client";
import { useAvailableIntegrations, useConnectedIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { usePermissions } from "@/contexts/usePermissions";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { toTestId } from "@/lib/testID";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
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
const CONNECT_OPTION_VALUE = "__connect_new__";

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

function getEmptyPlaceholder(integrationType: string | undefined): string {
  const integrationTypeLabel = integrationType ? getIntegrationTypeDisplayName(undefined, integrationType) : "";
  return integrationTypeLabel ? `No ${integrationTypeLabel} integrations available` : "No integrations available";
}

function IntegrationPickerEmptyState({
  field,
  integrationType,
}: {
  field: ConfigurationField;
  integrationType: string | undefined;
}) {
  return (
    <div data-testid={toTestId(`integration-field-${field.name}`)} className="space-y-2">
      <Select value="" disabled>
        <SelectTrigger className="w-full">
          <SelectValue placeholder={getEmptyPlaceholder(integrationType)} />
        </SelectTrigger>
      </Select>
      <p className="text-xs text-gray-500 dark:text-gray-400">Connect an integration in Organization settings first.</p>
    </div>
  );
}

function IntegrationPickerOptions({
  isRequired,
  options,
  canConnect,
}: {
  isRequired: boolean;
  options: OrganizationsIntegration[];
  canConnect: boolean;
}) {
  return (
    <>
      {!isRequired ? <SelectItem value={CLEAR_OPTION_VALUE}>None</SelectItem> : null}
      {options.map((integration) => {
        const name = getInstallationName(integration);
        return (
          <SelectItem key={name} value={name}>
            <IntegrationOptionLabel integration={integration} />
          </SelectItem>
        );
      })}
      {canConnect ? (
        <>
          {options.length > 0 ? <SelectSeparator /> : null}
          <SelectItem value={CONNECT_OPTION_VALUE} data-testid="integration-picker-connect-option">
            <span className="flex items-center gap-1.5 text-blue-600 dark:text-blue-400">
              <Plus className="size-3" />
              Connect an integration
            </span>
          </SelectItem>
        </>
      ) : null}
    </>
  );
}

/**
 * Inline connect flow for type-filtered fields, wired the same way the
 * component sidebar wires IntegrationCreateDialog for its "+ Connect another
 * instance" option. Unfiltered fields never open this dialog (there is no
 * single integration type to create).
 */
function ConnectIntegrationDialog({
  open,
  onOpenChange,
  organizationId,
  integrationType,
  enabled,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  organizationId: string;
  integrationType: string | undefined;
  enabled: boolean;
  onCreated: (installationName: string) => void;
}) {
  const canConnectInline = Boolean(integrationType) && enabled;
  const { data: availableIntegrationDefinitions = [] } = useAvailableIntegrations({ enabled: canConnectInline });
  const createIntegrationMutation = useCreateIntegration(organizationId, "node_configuration");
  const integrationDefinition = useMemo(
    () => availableIntegrationDefinitions.find((definition) => definition.name === integrationType),
    [availableIntegrationDefinitions, integrationType],
  );

  if (!canConnectInline) {
    return null;
  }

  return (
    <IntegrationCreateDialog
      open={open}
      onOpenChange={onOpenChange}
      integrationDefinition={integrationDefinition}
      organizationId={organizationId}
      onCreateIntegration={async (payload) => {
        const result = await createIntegrationMutation.mutateAsync(payload);
        return result.data;
      }}
      onReset={() => createIntegrationMutation.reset()}
      defaultName={integrationDefinition?.name ?? ""}
      integrationHomeHref={`/${organizationId}/settings/integrations`}
      onCreated={(_integrationId, instanceName) => onCreated(instanceName)}
    />
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
  const { canAct } = usePermissions();
  const canConnectIntegrations = canAct("integrations", "create");
  const [isConnectDialogOpen, setIsConnectDialogOpen] = useState(false);
  const { data: integrations = [], isLoading, error } = useConnectedIntegrations(organizationId);

  // Type-filtered fields connect the provider in place; unfiltered fields have
  // no single provider to create, so they open integration settings in a new
  // tab and the canvas stays put.
  const openConnectFlow = () => {
    if (integrationType) {
      setIsConnectDialogOpen(true);
      return;
    }
    window.open(`/${organizationId}/settings/integrations`, "_blank", "noopener,noreferrer");
  };

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

  if (options.length === 0 && !canConnectIntegrations) {
    return <IntegrationPickerEmptyState field={field} integrationType={integrationType} />;
  }

  const placeholder = isRequired ? (field.placeholder ?? "Select integration") : "None";

  return (
    <div data-testid={toTestId(`integration-field-${field.name}`)}>
      <Select
        value={selectedName || (isRequired ? "" : CLEAR_OPTION_VALUE)}
        onValueChange={(nextValue) => {
          if (nextValue === CONNECT_OPTION_VALUE) {
            openConnectFlow();
            return;
          }

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
          <IntegrationPickerOptions isRequired={isRequired} options={options} canConnect={canConnectIntegrations} />
        </SelectContent>
      </Select>
      <ConnectIntegrationDialog
        open={isConnectDialogOpen}
        onOpenChange={setIsConnectDialogOpen}
        organizationId={organizationId}
        integrationType={integrationType}
        enabled={canConnectIntegrations}
        onCreated={(installationName) => onChange({ name: installationName })}
      />
    </div>
  );
}
