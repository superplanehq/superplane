import type { ComponentsIntegrationRef, OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectSeparator, SelectTrigger, SelectValue } from "@/components/ui/select";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { SimpleTooltip } from "./SimpleTooltip";
import {
  CONNECT_ANOTHER_INSTANCE_VALUE,
  REQUIRED_FIELD_BADGE_CLASS,
  SETTINGS_TAB_DIVIDER_CLASS,
} from "./settingsTabConstants";

type SettingsTabIntegrationSectionProps = {
  allowCreateIntegrations: boolean;
  allowIntegrations: boolean;
  allowUpdateIntegrations: boolean;
  integrationDefinition?: { name?: string; label?: string; icon?: string };
  integrationName?: string;
  integrationsOfType: OrganizationsIntegration[];
  isFormDisabled: boolean;
  onOpenConfigureIntegrationDialog?: (integrationId: string) => void;
  onOpenCreateIntegrationDialog?: () => void;
  onSelectIntegration: (integration: ComponentsIntegrationRef | undefined) => void;
  requestAutosave: () => void;
  selectedIntegration?: ComponentsIntegrationRef;
  selectedIntegrationFull?: OrganizationsIntegration;
  showValidation: boolean;
  validationErrors: Set<string>;
};

export function SettingsTabIntegrationSection({
  allowCreateIntegrations,
  allowIntegrations,
  allowUpdateIntegrations,
  integrationDefinition,
  integrationName,
  integrationsOfType,
  isFormDisabled,
  onOpenConfigureIntegrationDialog,
  onOpenCreateIntegrationDialog,
  onSelectIntegration,
  requestAutosave,
  selectedIntegration,
  selectedIntegrationFull,
  showValidation,
  validationErrors,
}: SettingsTabIntegrationSectionProps) {
  if (!integrationName) {
    return null;
  }

  return (
    <div className={SETTINGS_TAB_DIVIDER_CLASS}>
      {!allowIntegrations ? (
        <div className="bg-gray-50 dark:bg-gray-900/30 border border-gray-200 dark:border-gray-700 rounded-md p-3 text-sm text-gray-600 dark:text-gray-300">
          You don't have permission to view integrations.
        </div>
      ) : integrationsOfType.length === 0 ? (
        <SettingsTabIntegrationConnectPrompt
          allowCreateIntegrations={allowCreateIntegrations}
          integrationDefinition={integrationDefinition}
          integrationName={integrationName}
          isFormDisabled={isFormDisabled}
          onOpenCreateIntegrationDialog={onOpenCreateIntegrationDialog}
        />
      ) : (
        <>
          <div className="flex flex-col gap-2">
            <Label className="min-w-[100px] text-left">
              Integration
              <span className="text-gray-800 ml-1">*</span>
              {showValidation && validationErrors.has("integration") && (
                <span className={REQUIRED_FIELD_BADGE_CLASS}>Required</span>
              )}
            </Label>
            <p className="text-xs text-gray-500">Instance</p>
            <Select
              value={selectedIntegration?.id || ""}
              onValueChange={(value) => {
                if (isFormDisabled) {
                  return;
                }
                if (value === CONNECT_ANOTHER_INSTANCE_VALUE) {
                  if (allowCreateIntegrations && onOpenCreateIntegrationDialog) {
                    onOpenCreateIntegrationDialog();
                  }
                  return;
                }
                const integration = integrationsOfType.find((item) => item.metadata?.id === value);
                if (integration) {
                  onSelectIntegration({
                    id: integration.metadata?.id,
                    name: integration.metadata?.name,
                  });
                  requestAutosave();
                }
              }}
              disabled={isFormDisabled}
            >
              <SelectTrigger className="w-full shadow-none">
                <SelectValue placeholder="Select an installation" />
              </SelectTrigger>
              <SelectContent>
                {integrationsOfType.map((integration) => {
                  const instanceName = integration.metadata?.name;
                  const typeName = integration.metadata?.integrationName;
                  const displayName =
                    instanceName?.toLowerCase() === typeName?.toLowerCase()
                      ? getIntegrationTypeDisplayName(undefined, typeName) || instanceName
                      : instanceName;
                  return (
                    <SelectItem key={integration.metadata?.id} value={integration.metadata?.id || ""}>
                      {displayName || "Unnamed integration"}
                    </SelectItem>
                  );
                })}
                {onOpenCreateIntegrationDialog && allowCreateIntegrations && !isFormDisabled && (
                  <>
                    <SelectSeparator />
                    <SelectItem value={CONNECT_ANOTHER_INSTANCE_VALUE}>+ Connect another instance</SelectItem>
                  </>
                )}
              </SelectContent>
            </Select>
          </div>
          {selectedIntegrationFull ? (
            <>
              <p className="py-2 text-xs text-gray-500">Connection</p>
              <SettingsTabIntegrationStatusCard
                allowUpdateIntegrations={allowUpdateIntegrations}
                integrationDefinition={integrationDefinition}
                isFormDisabled={isFormDisabled}
                onOpenConfigureIntegrationDialog={onOpenConfigureIntegrationDialog}
                selectedIntegrationFull={selectedIntegrationFull}
              />
            </>
          ) : null}
        </>
      )}
    </div>
  );
}

function SettingsTabIntegrationStatusCard({
  allowUpdateIntegrations,
  integrationDefinition,
  isFormDisabled,
  onOpenConfigureIntegrationDialog,
  selectedIntegrationFull,
}: {
  allowUpdateIntegrations: boolean;
  integrationDefinition?: { name?: string; label?: string; icon?: string };
  isFormDisabled: boolean;
  onOpenConfigureIntegrationDialog?: (integrationId: string) => void;
  selectedIntegrationFull: OrganizationsIntegration;
}) {
  const hasIntegrationError =
    selectedIntegrationFull.status?.state === "error" && !!selectedIntegrationFull.status?.stateDescription;

  const integrationStatusCard = (
    <div
      className={`border border-gray-300 dark:border-gray-700 rounded-md bg-stripe-diagonal p-3 flex items-center justify-between gap-4 ${
        selectedIntegrationFull.status?.state === "ready"
          ? "bg-green-100 dark:bg-green-950/30"
          : selectedIntegrationFull.status?.state === "error"
            ? "bg-red-100 dark:bg-red-950/30"
            : "bg-orange-100 dark:bg-orange-950/30"
      }`}
    >
      <div className="flex items-center gap-2 min-w-0">
        <IntegrationIcon
          integrationName={selectedIntegrationFull.metadata?.integrationName}
          iconSlug={integrationDefinition?.icon}
          className="mt-0.5 h-4 w-4 flex-shrink-0 text-gray-500 dark:text-gray-400"
        />
        <div className="min-w-0">
          <h3 className="text-sm font-semibold text-gray-800 dark:text-gray-100 truncate">
            {getIntegrationTypeDisplayName(undefined, selectedIntegrationFull.metadata?.integrationName) ||
              "Integration"}
          </h3>
        </div>
      </div>
      <div className="flex items-center gap-2 flex-shrink-0">
        <span
          className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
            selectedIntegrationFull.status?.state === "ready"
              ? "border border-green-950/15 bg-green-100 text-green-800 dark:border-green-950/15 dark:bg-green-900/30 dark:text-green-400"
              : selectedIntegrationFull.status?.state === "error"
                ? "border border-red-950/15 bg-red-100 text-red-800 dark:border-red-950/15 dark:bg-red-900/30 dark:text-red-400"
                : "border border-orange-950/15 bg-orange-100 text-yellow-800 dark:border-orange-950/15 dark:bg-orange-950/30 dark:text-yellow-400"
          }`}
        >
          {selectedIntegrationFull.status?.state
            ? selectedIntegrationFull.status.state.charAt(0).toUpperCase() +
              selectedIntegrationFull.status.state.slice(1)
            : "Unknown"}
        </span>
        {selectedIntegrationFull.metadata?.id && onOpenConfigureIntegrationDialog && !isFormDisabled ? (
          <Button
            variant="outline"
            size="sm"
            className="text-sm py-1.5"
            onClick={() => onOpenConfigureIntegrationDialog(selectedIntegrationFull.metadata!.id!)}
            disabled={!allowUpdateIntegrations}
          >
            Configure...
          </Button>
        ) : null}
      </div>
    </div>
  );

  if (hasIntegrationError) {
    return (
      <SimpleTooltip content={selectedIntegrationFull.status?.stateDescription || ""} interactive={true}>
        {integrationStatusCard}
      </SimpleTooltip>
    );
  }

  return integrationStatusCard;
}

function SettingsTabIntegrationConnectPrompt({
  allowCreateIntegrations,
  integrationDefinition,
  integrationName,
  isFormDisabled,
  onOpenCreateIntegrationDialog,
}: {
  allowCreateIntegrations: boolean;
  integrationDefinition?: { name?: string; label?: string; icon?: string };
  integrationName: string;
  isFormDisabled: boolean;
  onOpenCreateIntegrationDialog?: () => void;
}) {
  return (
    <div className="bg-orange-100 dark:bg-orange-950/30 border border-orange-950/15 rounded-md bg-stripe-diagonal p-3 flex items-center justify-between gap-4">
      <div className="flex items-center gap-2 min-w-0">
        <IntegrationIcon
          integrationName={integrationName}
          iconSlug={integrationDefinition?.icon}
          className="h-4 w-4 flex-shrink-0 text-gray-500 dark:text-gray-400"
        />
        <span className="text-sm font-semibold text-gray-800 dark:text-gray-100 truncate">
          {getIntegrationTypeDisplayName(undefined, integrationName) || integrationName} Integration
        </span>
      </div>
      <Button
        variant="outline"
        size="sm"
        onClick={onOpenCreateIntegrationDialog}
        className="flex-shrink-0"
        disabled={!allowCreateIntegrations || isFormDisabled}
      >
        Connect
      </Button>
    </div>
  );
}
